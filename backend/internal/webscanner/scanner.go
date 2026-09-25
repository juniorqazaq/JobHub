package webscanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/net/html"
	"jobhub-ai/backend/internal/jobs"
)

const MaxRequests = 30
const MaxVacancies = 50
const MaxResponseBytes = 3 << 20

type Resolver interface {
	LookupIP(context.Context, string, string) ([]net.IP, error)
}
type netResolver struct{}

func (netResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

type Result struct {
	ID              string
	Domain          string
	CareerURL       string
	ATS             string
	ProviderKey     string
	ATSURL          string
	ATSConfidence   string
	DetectionSource string
	ScanMethod      string
	PagesLoaded     int
	BrowserRequests int
	StartedAt       time.Time
	CompletedAt     time.Time
	Status          string
	ErrorCategory   string
	RequestCount    int
	VacancyURLs     int
	Parsed          int
	Skipped         int
	Warnings        []string
	Items           []jobs.ImportedJob
}

type Scanner struct {
	client      *http.Client
	resolver    Resolver
	maxRequests int
	maxBytes    int64
	adapters    adapterRegistry
	browser     BrowserRenderer
}

type BrowserRenderer interface {
	Scan(context.Context, *url.URL, string) (BrowserResult, error)
}

type BrowserResult struct {
	FinalURL     string
	PagesLoaded  int
	RequestCount int
	VacancyURLs  int
	Skipped      int
	Warnings     []string
	Items        []jobs.ImportedJob
}

func New(client *http.Client) *Scanner {
	return newScanner(client, netResolver{}, MaxRequests, MaxResponseBytes)
}

// NewWithBrowser keeps the renderer optional and outside the normal HTTP path.
func NewWithBrowser(client *http.Client, browser BrowserRenderer) *Scanner {
	s := New(client)
	s.browser = browser
	return s
}
func newScanner(client *http.Client, resolver Resolver, maxRequests int, maxBytes int64) *Scanner {
	if client == nil {
		client = &http.Client{}
	}
	c := *client
	c.Timeout = 15 * time.Second
	if c.Transport == nil {
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
			host, port, err := net.SplitHostPort(address)
			if err != nil {
				return nil, scanError("TRANSPORT_FAILED")
			}
			ips, err := resolver.LookupIP(ctx, "ip", host)
			if err != nil || len(ips) == 0 {
				return nil, scanError("DNS_FAILED")
			}
			for _, ip := range ips {
				if !safeIP(ip) {
					return nil, scanError("UNSAFE_URL")
				}
			}
			return (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext(ctx, network, net.JoinHostPort(ips[0].String(), port))
		}
		c.Transport = transport
	}
	return &Scanner{client: &c, resolver: resolver, maxRequests: maxRequests, maxBytes: maxBytes, adapters: defaultRegistry(&c)}
}

var scanSequence atomic.Uint64

func scanID(now time.Time) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%d:%d", now.UnixNano(), scanSequence.Add(1))))
	return "ws_" + hex.EncodeToString(sum[:8])
}

func (s *Scanner) Scan(ctx context.Context, raw string) (out Result, err error) {
	out.StartedAt = time.Now().UTC()
	out.ID = scanID(out.StartedAt)
	out.Status = "failed"
	out.ScanMethod = "unsupported"
	defer func() { out.CompletedAt = time.Now().UTC() }()
	root, err := ValidateURL(ctx, raw, s.resolver)
	if err != nil {
		out.ErrorCategory = "UNSAFE_URL"
		return out, err
	}
	out.Domain = NormalizeDomain(root.Hostname())
	budget := &fetchBudget{remaining: s.maxRequests}
	home, final, err := s.fetch(ctx, root, budget)
	out.RequestCount = s.maxRequests - budget.remaining
	if err != nil {
		out.ErrorCategory = category(err)
		return out, err
	}
	links, detection := discover(final, home)
	out.ATS, out.ProviderKey, out.ATSURL, out.ATSConfidence, out.DetectionSource = detection.Provider, detection.ProviderKey, detection.URL, detection.Confidence, detection.Source
	if detection.Provider == "greenhouse" && detection.URL != "" {
		atsURL, _ := url.Parse(detection.URL)
		parts := strings.Split(strings.Trim(atsURL.Path, "/"), "/")
		if len(parts) > 0 && parts[0] != "" && budget.remaining > 0 {
			budget.remaining--
			if adapter := s.adapters["greenhouse"]; adapter != nil {
				collected, collectErr := adapter.Collect(ctx, parts[0], MaxVacancies)
				out.RequestCount = s.maxRequests - budget.remaining
				if collectErr == nil {
					out.CareerURL = atsURL.String()
					out.Items = collected.Items
					out.VacancyURLs = collected.Fetched
					out.Parsed = len(collected.Items)
					out.Skipped = collected.Skipped + collected.Malformed
					out.Status = "succeeded"
					out.ScanMethod = "ats"
					return out, nil
				}
			}
		}
	}
	if len(links) == 0 {
		sitemap := *final
		sitemap.Path = "/sitemap.xml"
		sitemap.RawQuery = ""
		sitemap.Fragment = ""
		if body, _, mapErr := s.fetch(ctx, &sitemap, budget); mapErr == nil {
			links = sitemapCareerLinks(final, body)
		}
		out.RequestCount = s.maxRequests - budget.remaining
	}
	career := final
	careerBody := home
	careerFetched := false
	for _, candidate := range links {
		body, candidateFinal, candidateErr := s.fetch(ctx, candidate, budget)
		out.RequestCount = s.maxRequests - budget.remaining
		if candidateErr != nil {
			continue
		}
		careerBody = body
		career = candidateFinal
		careerFetched = true
		break
	}
	if !careerFetched {
		for _, candidate := range commonCareerCandidates(final) {
			body, candidateFinal, candidateErr := s.fetch(ctx, candidate, budget)
			out.RequestCount = s.maxRequests - budget.remaining
			if candidateErr != nil {
				continue
			}
			careerBody = body
			career = candidateFinal
			careerFetched = true
			break
		}
	}
	if careerFetched {
		out.CareerURL = career.String()
	} else if detection.URL != "" {
		out.CareerURL = detection.URL
	}
	items, jobLinks, warnings := extractPage(career, careerBody, out.Domain)
	out.Warnings = warnings
	if len(items) == 0 {
		for _, jobURL := range jobLinks {
			if len(items) >= MaxVacancies {
				out.Warnings = append(out.Warnings, "vacancy_limit_reached")
				break
			}
			body, finalJob, fetchErr := s.fetch(ctx, jobURL, budget)
			out.RequestCount = s.maxRequests - budget.remaining
			if fetchErr != nil {
				out.Skipped++
				continue
			}
			parsed, _, warn := extractPage(finalJob, body, out.Domain)
			out.Warnings = append(out.Warnings, warn...)
			if len(parsed) == 0 {
				if item, ok := extractHTMLJob(finalJob, body, out.Domain); ok {
					parsed = []jobs.ImportedJob{item}
				}
			}
			if len(parsed) == 0 {
				out.Skipped++
				continue
			}
			items = append(items, parsed[0])
		}
	}
	out.VacancyURLs = len(jobLinks)
	if len(items) > out.VacancyURLs {
		out.VacancyURLs = len(items)
	}
	out.Items = dedupe(items)
	out.Parsed = len(out.Items)
	out.Skipped += len(items) - len(out.Items)
	out.Status = "succeeded"
	out.ErrorCategory = ""
	if out.Parsed > 0 {
		out.ScanMethod = "static"
		return out, nil
	}
	if s.browser != nil && looksLikeSPA(home, careerBody) {
		browserResult, browserErr := s.browser.Scan(ctx, career, out.Domain)
		out.PagesLoaded = browserResult.PagesLoaded
		out.BrowserRequests = browserResult.RequestCount
		out.Warnings = append(out.Warnings, browserResult.Warnings...)
		if browserErr != nil {
			out.Warnings = append(out.Warnings, strings.ToLower(category(browserErr)))
			return out, nil
		}
		out.CareerURL = browserResult.FinalURL
		out.VacancyURLs = browserResult.VacancyURLs
		out.Skipped += browserResult.Skipped
		out.Items = dedupe(browserResult.Items)
		out.Parsed = len(out.Items)
		if out.Parsed > 0 {
			out.ScanMethod = "browser"
		} else {
			out.Warnings = append(out.Warnings, "rendered_vacancies_not_found")
		}
	}
	return out, nil
}

func looksLikeSPA(pages ...[]byte) bool {
	for _, body := range pages {
		text := strings.ToLower(string(body))
		if strings.Contains(text, "<script") && (strings.Contains(text, `id="root"`) || strings.Contains(text, `id="app"`) || strings.Contains(text, "<app-root") || strings.Contains(text, "type=\"module\"") || strings.Contains(text, "type='module'")) {
			return true
		}
	}
	return false
}

type fetchBudget struct{ remaining int }
type scanError string

func (e scanError) Error() string { return string(e) }
func category(err error) string {
	var e scanError
	if errors.As(err, &e) {
		return string(e)
	}
	return "SCAN_FAILED"
}

func (s *Scanner) fetch(ctx context.Context, target *url.URL, budget *fetchBudget) ([]byte, *url.URL, error) {
	if budget.remaining <= 0 {
		return nil, nil, scanError("REQUEST_LIMIT")
	}
	budget.remaining--
	if _, err := ValidateURL(ctx, target.String(), s.resolver); err != nil {
		return nil, nil, scanError("UNSAFE_URL")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, nil, scanError("INVALID_URL")
	}
	req.Header.Set("User-Agent", "JobHub-Website-Scanner/1.0 (+https://jobhub.kz)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	client := *s.client
	client.CheckRedirect = func(r *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return scanError("TOO_MANY_REDIRECTS")
		}
		if budget.remaining <= 0 {
			return scanError("REQUEST_LIMIT")
		}
		budget.remaining--
		_, e := ValidateURL(r.Context(), r.URL.String(), s.resolver)
		return e
	}
	res, err := client.Do(req)
	if err != nil {
		if cat := category(err); cat != "SCAN_FAILED" {
			return nil, nil, scanError(cat)
		}
		return nil, nil, scanError("TRANSPORT_FAILED")
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, nil, scanError("HTTP_FAILED")
	}
	ct := strings.ToLower(res.Header.Get("Content-Type"))
	if ct != "" && !strings.Contains(ct, "text/html") && !strings.Contains(ct, "application/xhtml+xml") && !strings.Contains(ct, "xml") {
		return nil, nil, scanError("INVALID_CONTENT_TYPE")
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, s.maxBytes+1))
	if err != nil {
		return nil, nil, scanError("RESPONSE_READ_FAILED")
	}
	if int64(len(body)) > s.maxBytes {
		return nil, nil, scanError("RESPONSE_TOO_LARGE")
	}
	return body, res.Request.URL, nil
}

func ValidateURL(ctx context.Context, raw string, resolver Resolver) (*url.URL, error) {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil || u.User != nil || u.Hostname() == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, scanError("UNSAFE_URL")
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return nil, scanError("UNSAFE_URL")
	}
	if ip := net.ParseIP(host); ip != nil {
		if !safeIP(ip) {
			return nil, scanError("UNSAFE_URL")
		}
	} else {
		ips, err := resolver.LookupIP(ctx, "ip", host)
		if err != nil || len(ips) == 0 {
			return nil, scanError("DNS_FAILED")
		}
		for _, ip := range ips {
			if !safeIP(ip) {
				return nil, scanError("UNSAFE_URL")
			}
		}
	}
	u.Fragment = ""
	return u, nil
}
func safeIP(ip net.IP) bool {
	return !(ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified() || ip.IsMulticast())
}
func NormalizeDomain(host string) string {
	return strings.TrimPrefix(strings.ToLower(strings.TrimSuffix(host, ".")), "www.")
}

var careerTerms = []string{"career", "careers", "job", "jobs", "vacancy", "vacancies", "work-with-us", "join-us", "career-opportunities", "ваканс", "карьера", "работа", "работа у нас", "бос орындар", "вакансиялар", "мансап", "жұмыс"}

type Detection struct{ Provider, ProviderKey, URL, Confidence, Source string }
type atsPattern struct {
	provider string
	hosts    []string
	key      *regexp.Regexp
}

var atsPatterns = []atsPattern{
	{"greenhouse", []string{"greenhouse.io"}, regexp.MustCompile(`(?i)(?:boards\.greenhouse\.io|boards-api\.greenhouse\.io/v1/boards)/([a-z0-9_-]+)`)},
	{"lever", []string{"lever.co"}, regexp.MustCompile(`(?i)(?:jobs\.lever\.co|api\.lever\.co/v0/postings)/([a-z0-9_-]+)`)},
	{"workday", []string{"myworkdayjobs.com"}, regexp.MustCompile(`(?i)https?://([a-z0-9_-]+)\.wd\d+\.myworkdayjobs\.com`)},
	{"smartrecruiters", []string{"smartrecruiters.com"}, regexp.MustCompile(`(?i)(?:careers\.smartrecruiters\.com|api\.smartrecruiters\.com/v1/companies)/([a-z0-9_-]+)`)},
	{"ashby", []string{"ashbyhq.com"}, regexp.MustCompile(`(?i)(?:jobs\.ashbyhq\.com|api\.ashbyhq\.com/posting-api/job-board)/([a-z0-9_-]+)`)},
	{"recruitee", []string{"recruitee.com"}, regexp.MustCompile(`(?i)https?://([a-z0-9_-]+)\.recruitee\.com`)},
	{"workable", []string{"workable.com"}, regexp.MustCompile(`(?i)(?:apply|jobs)\.workable\.com/([a-z0-9_-]+)`)},
	{"bamboohr", []string{"bamboohr.com"}, regexp.MustCompile(`(?i)https?://([a-z0-9_-]+)\.bamboohr\.com`)},
	{"teamtailor", []string{"teamtailor.com"}, regexp.MustCompile(`(?i)https?://([a-z0-9_-]+)\.teamtailor\.com`)},
	{"personio", []string{"personio.com", "personio.de"}, regexp.MustCompile(`(?i)https?://([a-z0-9_-]+)\.jobs\.personio\.(?:com|de)`)},
}

func discover(base *url.URL, body []byte) ([]*url.URL, Detection) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, Detection{}
	}
	type candidate struct {
		u     *url.URL
		score int
	}
	var rows []candidate
	detection := Detection{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			for _, key := range []string{"href", "src", "action"} {
				raw := attr(n, key)
				if raw == "" {
					continue
				}
				u, parseErr := base.Parse(raw)
				if parseErr == nil && detection.Provider == "" {
					if found, ok := detectATSURL(u.String(), key); ok {
						detection = found
					}
				}
			}
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			href := attr(n, "href")
			u, err := base.Parse(href)
			if err == nil && (u.Scheme == "http" || u.Scheme == "https") {
				text := strings.ToLower(nodeText(n))
				hay := strings.ToLower(u.Path + " " + text)
				score := 0
				for _, term := range careerTerms {
					if strings.Contains(hay, term) {
						score++
					}
				}
				if score > 0 && NormalizeDomain(u.Hostname()) == NormalizeDomain(base.Hostname()) {
					u.Fragment = ""
					// Query parameters on career landing links are commonly filters.
					// Start from the canonical listing so one category cannot consume the scan.
					if isListingPath(u.Path) {
						u.RawQuery = ""
					}
					rows = append(rows, candidate{u, score})
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].score > rows[j].score })
	seen := map[string]bool{}
	out := []*url.URL{}
	for _, r := range rows {
		key := r.u.String()
		if !seen[key] && len(out) < 10 {
			seen[key] = true
			out = append(out, r.u)
		}
	}
	return out, detection
}
func detectATSURL(raw, source string) (Detection, bool) {
	lower := strings.ToLower(raw)
	for _, p := range atsPatterns {
		matched := false
		for _, host := range p.hosts {
			if strings.Contains(lower, host) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		d := Detection{Provider: p.provider, URL: raw, Confidence: "medium", Source: source}
		if match := p.key.FindStringSubmatch(raw); len(match) > 1 && match[1] != "" {
			d.ProviderKey = match[1]
			d.Confidence = "high"
		}
		return d, true
	}
	return Detection{}, false
}

func commonCareerCandidates(base *url.URL) []*url.URL {
	paths := []string{"/career", "/careers", "/job", "/jobs", "/vacancy", "/vacancies", "/about/career", "/about/careers", "/company/careers", "/work-with-us", "/join-us"}
	out := make([]*url.URL, 0, len(paths)+2)
	for _, path := range paths {
		u := *base
		u.Path = path
		u.RawQuery = ""
		u.Fragment = ""
		out = append(out, &u)
	}
	for _, prefix := range []string{"careers.", "jobs."} {
		u := *base
		u.Host = prefix + base.Hostname()
		if port := base.Port(); port != "" {
			u.Host = net.JoinHostPort(prefix+base.Hostname(), port)
		}
		u.Path = "/"
		u.RawQuery = ""
		u.Fragment = ""
		out = append(out, &u)
	}
	return out
}

func sitemapCareerLinks(base *url.URL, body []byte) []*url.URL {
	text := string(body)
	out := []*url.URL{}
	for {
		start := strings.Index(strings.ToLower(text), "<loc>")
		if start < 0 {
			break
		}
		text = text[start+5:]
		end := strings.Index(strings.ToLower(text), "</loc>")
		if end < 0 {
			break
		}
		raw := strings.TrimSpace(text[:end])
		text = text[end+6:]
		u, err := base.Parse(raw)
		if err != nil || NormalizeDomain(u.Hostname()) != NormalizeDomain(base.Hostname()) {
			continue
		}
		hay := strings.ToLower(u.Path)
		for _, term := range careerTerms {
			if strings.Contains(hay, term) {
				u.Fragment = ""
				out = append(out, u)
				break
			}
		}
		if len(out) >= 10 {
			break
		}
	}
	return uniqueURLs(out)
}

func extractPage(base *url.URL, body []byte, domain string) ([]jobs.ImportedJob, []*url.URL, []string) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return nil, nil, []string{"malformed_html"}
	}
	items := []jobs.ImportedJob{}
	links := []*url.URL{}
	warnings := []string{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" && strings.EqualFold(attr(n, "type"), "application/ld+json") {
			parsed, err := parseJSONLD([]byte(nodeText(n)), base, domain)
			if err != nil {
				warnings = append(warnings, "malformed_jsonld")
			} else {
				items = append(items, parsed...)
			}
		}
		if n.Type == html.ElementNode && n.Data == "a" {
			u, err := base.Parse(attr(n, "href"))
			if err == nil && NormalizeDomain(u.Hostname()) == NormalizeDomain(base.Hostname()) {
				if isJobDetailPath(u.Path) {
					u.Fragment = ""
					links = append(links, u)
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return items, uniqueURLs(links), warnings
}

func isListingPath(path string) bool {
	p := strings.TrimSuffix(strings.ToLower(path), "/")
	return p == "/search" || p == "/career" || p == "/careers" || p == "/jobs" || p == "/vacancies" || p == "/vacancy"
}

func isJobDetailPath(path string) bool {
	p := strings.Trim(strings.ToLower(path), "/")
	parts := strings.Split(p, "/")
	for i, part := range parts {
		if (part == "job" || part == "jobs" || part == "vacancy" || part == "vacancies") && i+1 < len(parts) && strings.TrimSpace(parts[i+1]) != "" {
			return true
		}
	}
	return false
}

func extractHTMLJob(base *url.URL, body []byte, domain string) (jobs.ImportedJob, bool) {
	doc, err := html.Parse(strings.NewReader(string(body)))
	if err != nil {
		return jobs.ImportedJob{}, false
	}
	title := ""
	description := ""
	company := ""
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode {
			switch n.Data {
			case "h1":
				if title == "" {
					title = nodeText(n)
				}
			case "main", "article":
				if description == "" {
					description = nodeText(n)
				}
			case "meta":
				property := strings.ToLower(attr(n, "property"))
				name := strings.ToLower(attr(n, "name"))
				if company == "" && (property == "og:site_name" || name == "application-name") {
					company = attr(n, "content")
				}
				if title == "" && property == "og:title" {
					title = attr(n, "content")
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	title = strings.TrimSpace(title)
	description = strings.TrimSpace(description)
	if title == "" || description == "" {
		return jobs.ImportedJob{}, false
	}
	u := *base
	u.Fragment = ""
	return jobs.ImportedJob{Source: "website:" + domain, ExternalID: u.String(), SourceURL: u.String(), Title: title, CompanyNameRaw: strings.TrimSpace(company), Description: description, DescriptionKind: "full"}, true
}

func parseJSONLD(data []byte, base *url.URL, domain string) ([]jobs.ImportedJob, error) {
	var raw any
	if json.Unmarshal(data, &raw) != nil {
		return nil, errors.New("invalid jsonld")
	}
	nodes := flatten(raw)
	items := []jobs.ImportedJob{}
	for _, node := range nodes {
		if !isJobPosting(node["@type"]) {
			continue
		}
		title := textValue(node["title"])
		if title == "" {
			continue
		}
		jobURL := textValue(node["url"])
		if jobURL == "" {
			jobURL = base.String()
		}
		u, err := base.Parse(jobURL)
		if err != nil || u.Hostname() == "" {
			continue
		}
		u.Fragment = ""
		company := ""
		if org, ok := node["hiringOrganization"].(map[string]any); ok {
			company = textValue(org["name"])
		}
		location := locationValue(node["jobLocation"])
		salary := salaryValue(node["baseSalary"])
		description := plainHTML(textValue(node["description"]))
		employment := textValue(node["employmentType"])
		published := textValue(node["datePosted"])
		items = append(items, jobs.ImportedJob{Source: "website:" + domain, ExternalID: u.String(), SourceURL: u.String(), Title: title, CompanyNameRaw: company, LocationRaw: location, SalaryRaw: salary, EmploymentTypeRaw: employment, Description: description, DescriptionKind: "full", ExternalUpdatedRaw: published})
	}
	return items, nil
}
func flatten(v any) []map[string]any {
	out := []map[string]any{}
	switch x := v.(type) {
	case []any:
		for _, e := range x {
			out = append(out, flatten(e)...)
		}
	case map[string]any:
		if g, ok := x["@graph"]; ok {
			out = append(out, flatten(g)...)
		}
		out = append(out, x)
	}
	return out
}
func isJobPosting(v any) bool {
	switch x := v.(type) {
	case string:
		return strings.EqualFold(x, "JobPosting")
	case []any:
		for _, e := range x {
			if isJobPosting(e) {
				return true
			}
		}
	}
	return false
}
func textValue(v any) string {
	switch x := v.(type) {
	case string:
		return strings.TrimSpace(x)
	case []any:
		vals := []string{}
		for _, e := range x {
			if s := textValue(e); s != "" {
				vals = append(vals, s)
			}
		}
		return strings.Join(vals, ", ")
	}
	return ""
}
func locationValue(v any) string {
	if a, ok := v.([]any); ok {
		vals := []string{}
		for _, x := range a {
			if s := locationValue(x); s != "" {
				vals = append(vals, s)
			}
		}
		return strings.Join(vals, ", ")
	}
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	if addr, ok := m["address"].(map[string]any); ok {
		vals := []string{}
		for _, k := range []string{"addressLocality", "addressRegion", "addressCountry"} {
			if s := textValue(addr[k]); s != "" {
				vals = append(vals, s)
			}
		}
		return strings.Join(vals, ", ")
	}
	return textValue(m["name"])
}
func salaryValue(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	currency := textValue(m["currency"])
	value := m["value"]
	if vm, ok := value.(map[string]any); ok {
		return strings.TrimSpace(fmt.Sprint(vm["minValue"]) + "–" + fmt.Sprint(vm["maxValue"]) + " " + currency + " " + textValue(vm["unitText"]))
	}
	return strings.TrimSpace(fmt.Sprint(value) + " " + currency)
}
func plainHTML(s string) string {
	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		return ""
	}
	return strings.Join(strings.Fields(nodeText(doc)), " ")
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
func attr(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return strings.TrimSpace(a.Val)
		}
	}
	return ""
}
func uniqueURLs(in []*url.URL) []*url.URL {
	seen := map[string]bool{}
	out := []*url.URL{}
	for _, u := range in {
		if !seen[u.String()] {
			seen[u.String()] = true
			out = append(out, u)
		}
	}
	return out
}
func dedupe(in []jobs.ImportedJob) []jobs.ImportedJob {
	seen := map[string]bool{}
	out := []jobs.ImportedJob{}
	for _, x := range in {
		if x.ExternalID != "" && !seen[x.ExternalID] {
			seen[x.ExternalID] = true
			out = append(out, x)
		}
	}
	return out
}
