package telegramcollector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"jobhub-ai/backend/internal/webscanner"
)

type WebsiteScanner interface {
	Scan(context.Context, string) (webscanner.Result, error)
}
type Sender interface {
	SendMessage(context.Context, int64, string) error
}

type CommandProcessor struct {
	admins  map[int64]struct{}
	scanner WebsiteScanner
	sender  Sender
	logger  *slog.Logger
	mu      sync.Mutex
	scans   map[string]webscanner.Result
}

func NewCommandProcessor(admins map[int64]struct{}, scanner WebsiteScanner, sender Sender, logger *slog.Logger) *CommandProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	return &CommandProcessor{admins: admins, scanner: scanner, sender: sender, logger: logger, scans: map[string]webscanner.Result{}}
}
func (p *CommandProcessor) Process(ctx context.Context, chatID, userID int64, text, language string) (bool, error) {
	fields := strings.Fields(strings.TrimSpace(text))
	if len(fields) == 0 || !strings.HasPrefix(fields[0], "/") {
		return false, nil
	}
	cmd := strings.ToLower(strings.Split(fields[0], "@")[0])
	if !knownCommand(cmd) {
		return false, nil
	}
	if cmd == "/start" {
		return true, p.sender.SendMessage(ctx, chatID, start(language, userID))
	}
	if _, ok := p.admins[userID]; !ok {
		return true, p.sender.SendMessage(ctx, chatID, denied(language))
	}
	switch cmd {
	case "/help":
		return true, p.sender.SendMessage(ctx, chatID, help(language))
	case "/status":
		return true, p.sender.SendMessage(ctx, chatID, status(language, len(p.cachedScans())))
	case "/scan":
		if len(fields) != 2 {
			return true, p.sender.SendMessage(ctx, chatID, usageScan(language))
		}
		result, err := p.scanner.Scan(ctx, fields[1])
		if err != nil {
			p.logger.Warn("website scan failed", "category", result.ErrorCategory, "requests", result.RequestCount)
			return true, p.sender.SendMessage(ctx, chatID, scanFailed(language, result.ErrorCategory, result.RequestCount))
		}
		p.mu.Lock()
		p.scans[result.ID] = result
		p.mu.Unlock()
		return true, p.sender.SendMessage(ctx, chatID, scanSummary(language, result))
	}
	return false, nil
}
func knownCommand(c string) bool {
	switch c {
	case "/start", "/help", "/status", "/scan":
		return true
	}
	return false
}
func (p *CommandProcessor) cachedScans() []webscanner.Result {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]webscanner.Result, 0, len(p.scans))
	for _, r := range p.scans {
		if time.Since(r.CompletedAt) < 24*time.Hour {
			out = append(out, r)
		}
	}
	return out
}
func ru(lang string) bool { return strings.HasPrefix(strings.ToLower(lang), "ru") }
func denied(lang string) string {
	if ru(lang) {
		return "Доступ запрещён."
	}
	return "Қолжетімділікке рұқсат жоқ."
}
func help(lang string) string {
	if ru(lang) {
		return "Команды:\n/scan <url> — сухое сканирование\n/status — состояние"
	}
	return "Командалар:\n/scan <url> — dry-run сканерлеу\n/status — күйі"
}
func start(lang string, userID int64) string {
	if ru(lang) {
		return fmt.Sprintf("Ваш Telegram ID: %d\nДля сканирования администратор должен добавить его в TELEGRAM_ALLOWED_ADMIN_IDS.", userID)
	}
	return fmt.Sprintf("Telegram ID: %d\nСканерлеу үшін әкімші оны TELEGRAM_ALLOWED_ADMIN_IDS тізіміне қосуы керек.", userID)
}
func status(lang string, n int) string {
	if ru(lang) {
		return fmt.Sprintf("Сканер работает. Сохранено результатов: %d", n)
	}
	return fmt.Sprintf("Сканер жұмыс істеп тұр. Сақталған нәтиже: %d", n)
}
func usageScan(lang string) string {
	if ru(lang) {
		return "Использование: /scan https://company.kz"
	}
	return "Қолдану: /scan https://company.kz"
}
func scanFailed(lang, cat string, n int) string {
	if ru(lang) {
		return fmt.Sprintf("Сканирование не выполнено. Категория: %s\nЗапросов: %d", cat, n)
	}
	return fmt.Sprintf("Сканерлеу орындалмады. Санат: %s\nСұрау: %d", cat, n)
}
func scanSummary(lang string, r webscanner.Result) string {
	career := r.CareerURL
	if career == "" {
		career = "—"
	}
	ats := r.ATS
	if ats == "" {
		ats = "—"
	}
	sample := []string{}
	for i, x := range r.Items {
		if i == 3 {
			break
		}
		sample = append(sample, "• "+x.Title)
	}
	if ru(lang) {
		return fmt.Sprintf("Сканирование завершено\n\nСайт: %s\nCareer: %s\nМетод: %s\nПровайдер: %s\nHTTP-запросов: %d\nСтраниц браузера: %d\nЗапросов браузера: %d\nНайдено вакансий: %d\nРаспознано: %d\nПропущено: %d\n\n%s\n\nDry-run: в JobHub ничего не записано.", r.Domain, career, r.ScanMethod, ats, r.RequestCount, r.PagesLoaded, r.BrowserRequests, r.VacancyURLs, r.Parsed, r.Skipped, strings.Join(sample, "\n"))
	}
	return fmt.Sprintf("Сканерлеу аяқталды\n\nСайт: %s\nCareer: %s\nӘдіс: %s\nПровайдер: %s\nHTTP сұрауы: %d\nБраузер беті: %d\nБраузер сұрауы: %d\nВакансия табылды: %d\nТанылды: %d\nӨткізілді: %d\n\n%s\n\nDry-run: JobHub-қа ештеңе қосылған жоқ.", r.Domain, career, r.ScanMethod, ats, r.RequestCount, r.PagesLoaded, r.BrowserRequests, r.VacancyURLs, r.Parsed, r.Skipped, strings.Join(sample, "\n"))
}
