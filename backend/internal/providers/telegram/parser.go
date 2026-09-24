package telegram

import (
	"context"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers"
)

var labeledLine = regexp.MustCompile(`(?i)^\s*([\p{L} ]{2,24})\s*[-:–—]\s*(.+?)\s*$`)

var labels = map[string]string{
	"вакансия": "title", "должность": "title", "лауазым": "title", "бос орын": "title",
	"компания": "company", "работодатель": "company", "жұмыс беруші": "company",
	"город": "location", "место": "location", "локация": "location", "қала": "location", "орналасқан жері": "location",
	"зарплата": "salary", "оклад": "salary", "жалақы": "salary", "еңбекақы": "salary",
}

// Parse extracts only explicitly labelled values. The full original text is
// always retained as the description; absent labels remain empty/NULL.
func Parse(text string) Parsed {
	var out Parsed
	for _, line := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		match := labeledLine.FindStringSubmatch(line)
		if len(match) != 3 {
			continue
		}
		value := clean(match[2])
		if value == "" {
			continue
		}
		switch labels[strings.ToLower(strings.TrimSpace(match[1]))] {
		case "title":
			if out.Title == "" {
				out.Title = value
			}
		case "company":
			if out.Company == "" {
				out.Company = value
			}
		case "location":
			if out.Location == "" {
				out.Location = value
			}
		case "salary":
			if out.Salary == "" {
				out.Salary = value
			}
		}
	}
	return out
}

func Normalize(post Post) (jobs.ImportedJob, SkipReason) {
	text := clean(post.Text)
	if text == "" {
		return jobs.ImportedJob{}, SkipEmptyText
	}
	parsedURL, err := url.Parse(post.OriginalURL)
	if err != nil || parsedURL.Scheme != "https" || parsedURL.Hostname() != "t.me" || parsedURL.User != nil {
		return jobs.ImportedJob{}, SkipMissingSourceURL
	}
	parsed := Parse(text)
	if !meaningfulTitle(parsed.Title) {
		return jobs.ImportedJob{}, SkipMissingTitle
	}
	source := Source(post.ChatID)
	externalID := ExternalID(post.ChatID, post.MessageID)
	if post.ChatID == 0 || post.MessageID < 1 || post.TelegramDate.IsZero() || source == "" || externalID == "" {
		return jobs.ImportedJob{}, SkipValidationFailed
	}
	telegramDate := post.TelegramDate.UTC()
	return jobs.ImportedJob{
		Source: source, ExternalID: externalID, SourceURL: post.OriginalURL,
		UpstreamSourceName: post.ChannelTitle, CompanyNameRaw: parsed.Company,
		Title: parsed.Title, LocationRaw: parsed.Location, Description: text,
		DescriptionKind: "full", SalaryRaw: parsed.Salary,
		ExternalUpdatedAt: &telegramDate, ExternalUpdatedRaw: telegramDate.Format("2006-01-02T15:04:05Z"),
	}, ""
}

func Source(chatID int64) string {
	if chatID == 0 {
		return ""
	}
	return "telegram:" + strconv.FormatInt(chatID, 10)
}

func ExternalID(chatID, messageID int64) string {
	if chatID == 0 || messageID < 1 {
		return ""
	}
	return "telegram:" + strconv.FormatInt(chatID, 10) + ":" + strconv.FormatInt(messageID, 10)
}

func meaningfulTitle(value string) bool {
	return utf8.RuneCountInString(value) >= 3 && utf8.RuneCountInString(value) <= 200
}

func clean(value string) string { return strings.TrimSpace(strings.ReplaceAll(value, "\x00", "")) }

type Collection struct{ Item jobs.ImportedJob }

func (c Collection) Source() string { return c.Item.Source }
func (c Collection) Collect(_ context.Context) (providers.Result, error) {
	return providers.Result{Items: []jobs.ImportedJob{c.Item}, Fetched: 1, Complete: false}, nil
}
