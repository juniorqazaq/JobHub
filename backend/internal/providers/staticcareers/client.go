// Package staticcareers provides a bounded, configuration-driven adapter for
// public career pages whose vacancy details are available in ordinary HTML.
package staticcareers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
	"jobhub-ai/backend/internal/safety"
)

type Config struct{ Source, Company, ListingURL, DetailPrefix string }
type Client struct {
	cfg     Config
	http    *http.Client
	budget  *safety.RequestBudget
	maxJobs int
}

func New(cfg Config, maxRequests, maxJobs int, client *http.Client) (*Client, error) {
	if strings.TrimSpace(cfg.Source) == "" || strings.TrimSpace(cfg.ListingURL) == "" || maxRequests < 1 || maxRequests > 12 || maxJobs < 1 || maxJobs > 200 {
		return nil, providers.Failure("INVALID_BUDGET")
	}
	if client == nil {
		client = &http.Client{}
	}
	c := *client
	c.Timeout = 20 * time.Second
	c.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	b, _ := safety.NewRequestBudget(maxRequests)
	return &Client{cfg: cfg, http: &c, budget: b, maxJobs: maxJobs}, nil
}
func (c *Client) Source() string { return c.cfg.Source }
func (c *Client) Collect(ctx context.Context) (out providers.Result, err error) {
	defer func() {
		m := c.budget.Metrics()
		out.MaxRequests = m.MaxRequests
		out.RequestsUsed = m.RequestsUsed
		out.RemainingRequests = m.Remaining
	}()
	body, err := c.fetch(ctx, c.cfg.ListingURL, &out, false)
	if err != nil {
		return out, err
	}
	links := detailLinks(body, c.cfg.ListingURL, c.cfg.DetailPrefix)
	out.Fetched = len(links)
	if out.Fetched > c.maxJobs {
		return out, providers.Failure("JOB_BUDGET_EXHAUSTED")
	}
	out.PagesFetched = 1
	for _, link := range links {
		d, e := c.fetch(ctx, link, &out, true)
		if e != nil {
			out.Skipped++
			continue
		}
		item, ok := normalize(c.cfg, link, d)
		if !ok {
			out.Malformed++
			continue
		}
		out.Items = append(out.Items, item)
	}
	out.Complete = out.Skipped == 0 && out.Malformed == 0 && len(out.Items) == len(links)
	return out, nil
}
func (c *Client) fetch(ctx context.Context, raw string, out *providers.Result, detail bool) ([]byte, error) {
	if c.budget.Acquire() != nil {
		return nil, providers.Failure("REQUEST_BUDGET_EXHAUSTED")
	}
	out.Requests++
	if detail {
		out.DetailRequests++
	} else {
		out.ListRequests++
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	req.Header.Set("User-Agent", "JobHub-Career-POC/1.0")
	res, e := c.http.Do(req)
	if e != nil {
		return nil, providers.Failure("TRANSPORT_FAILED")
	}
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return nil, providers.Failure("HTTP_FAILED")
	}
	b, e := io.ReadAll(io.LimitReader(res.Body, 3<<20))
	if e != nil {
		return nil, providers.Failure("RESPONSE_READ_FAILED")
	}
	return b, nil
}
func detailLinks(body []byte, listing, prefix string) []string {
	base, _ := url.Parse(listing)
	seen := map[string]bool{}
	var out []string
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, a := range n.Attr {
				if a.Key == "href" {
					u, e := url.Parse(strings.TrimSpace(a.Val))
					if e == nil {
						u = base.ResolveReference(u)
						raw := u.String()
						if u.Host == base.Host && strings.HasPrefix(u.Path, prefix) && raw != listing && !seen[raw] {
							seen[raw] = true
							out = append(out, raw)
						}
					}
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			walk(ch)
		}
	}
	doc, e := html.Parse(strings.NewReader(string(body)))
	if e == nil {
		walk(doc)
	}
	return out
}

var locationPattern = regexp.MustCompile(`(?i)(Казахстан,\s*[А-ЯA-ZЁӘІҢҒҚӨҰҮҺа-яёәіңғқөұүһ ,/-]+(?:,\s*KZ)?)`)

func normalize(cfg Config, raw string, body []byte) (jobs.ImportedJob, bool) {
	doc, e := html.Parse(strings.NewReader(string(body)))
	if e != nil {
		return jobs.ImportedJob{}, false
	}
	var title, loc, desc string
	var text func(*html.Node)
	text = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				if title == "" {
					title = nodeText(n)
				}
			case "title":
				if title == "" {
					title = nodeText(n)
				}
			case "main", "article":
				if desc == "" {
					desc = nodeText(n)
				}
			}
		}
		for ch := n.FirstChild; ch != nil; ch = ch.NextSibling {
			text(ch)
		}
	}
	text(doc)
	if title == "" {
		title = "Vacancy"
	}
	if desc == "" {
		desc = nodeText(doc)
	}
	if match := locationPattern.FindString(desc); match != "" {
		loc = strings.TrimSpace(match)
	}
	title = strings.TrimSpace(title)
	if title == "" || title == "Vacancy" {
		return jobs.ImportedJob{}, false
	}
	sum := sha256.Sum256([]byte(raw))
	id := "url_" + hex.EncodeToString(sum[:8])
	return jobs.ImportedJob{Source: cfg.Source, ExternalID: id, SourceURL: raw, CompanyNameRaw: cfg.Company, Title: title, LocationRaw: strings.TrimSpace(loc), Description: strings.TrimSpace(desc), DescriptionKind: "full"}, true
}
func nodeText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(x *html.Node) {
		if x.Type == html.TextNode {
			b.WriteString(x.Data)
			b.WriteByte(' ')
		}
		for c := x.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return strings.Join(strings.Fields(b.String()), " ")
}
