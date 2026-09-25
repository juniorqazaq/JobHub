package webscanner

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"sync"
	"time"

	cdpbrowser "github.com/chromedp/cdproto/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/fetch"
	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/safety"
)

const (
	MaxBrowserPages    = 8
	MaxRenderedBytes   = 3 << 20
	MaxBrowserRequests = 100
	BrowserScanTimeout = 30 * time.Second
)

type ChromiumRenderer struct {
	resolver Resolver
	execPath string
	timeout  time.Duration
	maxPages int
	maxBytes int
}

func NewChromiumRenderer(execPath string) *ChromiumRenderer {
	return &ChromiumRenderer{resolver: netResolver{}, execPath: strings.TrimSpace(execPath), timeout: BrowserScanTimeout, maxPages: MaxBrowserPages, maxBytes: MaxRenderedBytes}
}

func (r *ChromiumRenderer) Scan(parent context.Context, target *url.URL, domain string) (out BrowserResult, err error) {
	if r == nil {
		return out, scanError("BROWSER_UNAVAILABLE")
	}
	ctx, cancel := context.WithTimeout(parent, r.timeout)
	defer cancel()
	requestBudget, _ := safety.NewRequestBudget(MaxBrowserRequests)
	defer func() {
		m := requestBudget.Metrics()
		out.RequestCount = m.RequestsUsed
		out.MaxRequests = m.MaxRequests
		out.RemainingRequests = m.Remaining
	}()
	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", true),
		chromedp.Flag("disable-extensions", true),
		chromedp.Flag("disable-background-networking", true),
		chromedp.Flag("disable-sync", true),
		chromedp.Flag("no-first-run", true),
	)
	if r.execPath != "" {
		opts = append(opts, chromedp.ExecPath(r.execPath))
	}
	allocator, cancelAllocator := chromedp.NewExecAllocator(ctx, opts...)
	defer cancelAllocator()
	browserCtx, cancelBrowser := chromedp.NewContext(allocator)
	defer cancelBrowser()

	var blockedErr error
	var blockedMu sync.Mutex
	chromedp.ListenTarget(browserCtx, func(event any) {
		paused, ok := event.(*fetch.EventRequestPaused)
		if !ok {
			return
		}
		go func() {
			executor := cdp.WithExecutor(browserCtx, chromedp.FromContext(browserCtx).Target)
			requestURL := paused.Request.URL
			resourceType := paused.ResourceType
			if resourceType == network.ResourceTypeImage || resourceType == network.ResourceTypeMedia || resourceType == network.ResourceTypeFont {
				_ = fetch.FailRequest(paused.RequestID, network.ErrorReasonBlockedByClient).Do(executor)
				return
			}
			if _, validateErr := ValidateURL(browserCtx, requestURL, r.resolver); validateErr != nil {
				blockedMu.Lock()
				if blockedErr == nil && resourceType == network.ResourceTypeDocument {
					blockedErr = scanError("UNSAFE_URL")
				}
				blockedMu.Unlock()
				_ = fetch.FailRequest(paused.RequestID, network.ErrorReasonBlockedByClient).Do(executor)
				return
			}
			if requestBudget.Acquire() != nil {
				_ = fetch.FailRequest(paused.RequestID, network.ErrorReasonBlockedByClient).Do(executor)
				return
			}
			_ = fetch.ContinueRequest(paused.RequestID).Do(executor)
		}()
	})
	if err := chromedp.Run(browserCtx,
		fetch.Enable().WithPatterns([]*fetch.RequestPattern{{URLPattern: "*"}}),
		cdpbrowser.SetDownloadBehavior(cdpbrowser.SetDownloadBehaviorBehaviorDeny),
	); err != nil {
		return out, scanError("BROWSER_START_FAILED")
	}

	load := func(raw string) ([]byte, string, error) {
		if out.PagesLoaded >= r.maxPages {
			return nil, "", scanError("PAGE_LIMIT")
		}
		validated, validateErr := ValidateURL(browserCtx, raw, r.resolver)
		if validateErr != nil {
			return nil, "", validateErr
		}
		var rendered string
		var finalURL string
		out.PagesLoaded++
		runErr := chromedp.Run(browserCtx,
			chromedp.Navigate(validated.String()),
			chromedp.Sleep(1200*time.Millisecond),
			chromedp.Location(&finalURL),
			chromedp.OuterHTML("html", &rendered, chromedp.ByQuery),
		)
		blockedMu.Lock()
		securityErr := blockedErr
		blockedErr = nil
		blockedMu.Unlock()
		if securityErr != nil {
			return nil, finalURL, securityErr
		}
		if runErr != nil {
			if errors.Is(ctx.Err(), context.DeadlineExceeded) {
				return nil, finalURL, scanError("BROWSER_TIMEOUT")
			}
			return nil, finalURL, scanError("BROWSER_NAVIGATION_FAILED")
		}
		if len(rendered) > r.maxBytes {
			return nil, finalURL, scanError("RENDERED_CONTENT_TOO_LARGE")
		}
		if _, validateErr = ValidateURL(browserCtx, finalURL, r.resolver); validateErr != nil {
			return nil, finalURL, validateErr
		}
		return []byte(rendered), finalURL, nil
	}

	body, finalURL, loadErr := load(target.String())
	if loadErr != nil {
		return out, loadErr
	}
	out.FinalURL = finalURL
	final, _ := url.Parse(finalURL)
	items, links, warnings := extractPage(final, body, domain)
	out.Warnings = append(out.Warnings, warnings...)
	if len(items) > 0 {
		out.Items = dedupe(items)
		out.VacancyURLs = len(out.Items)
		return out, nil
	}
	out.VacancyURLs = len(links)
	for _, jobURL := range links {
		if len(out.Items) >= MaxVacancies || out.PagesLoaded >= r.maxPages {
			break
		}
		detail, detailURL, detailErr := load(jobURL.String())
		if detailErr != nil {
			out.Skipped++
			continue
		}
		detailBase, _ := url.Parse(detailURL)
		parsed, _, detailWarnings := extractPage(detailBase, detail, domain)
		out.Warnings = append(out.Warnings, detailWarnings...)
		if len(parsed) == 0 {
			if item, ok := extractHTMLJob(detailBase, detail, domain); ok {
				parsed = []jobs.ImportedJob{item}
			}
		}
		if len(parsed) == 0 {
			out.Skipped++
			continue
		}
		out.Items = append(out.Items, parsed[0])
	}
	out.Items = dedupe(out.Items)
	return out, nil
}
