package telegramcollector

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"jobhub-ai/backend/internal/ingestion"
	"jobhub-ai/backend/internal/jobs"
	"jobhub-ai/backend/internal/providers/website"
	"jobhub-ai/backend/internal/webscanner"
)

type WebsiteScanner interface {
	Scan(context.Context, string) (webscanner.Result, error)
}
type Sender interface {
	SendMessage(context.Context, int64, string) error
}

type CommandProcessor struct {
	admins    map[int64]struct{}
	scanner   WebsiteScanner
	sender    Sender
	registrar SourceRegistrar
	store     jobs.Store
	logger    *slog.Logger
	mu        sync.Mutex
	scans     map[string]webscanner.Result
}

func NewCommandProcessor(admins map[int64]struct{}, scanner WebsiteScanner, sender Sender, registrar SourceRegistrar, store jobs.Store, logger *slog.Logger) *CommandProcessor {
	if logger == nil {
		logger = slog.Default()
	}
	return &CommandProcessor{admins: admins, scanner: scanner, sender: sender, registrar: registrar, store: store, logger: logger, scans: map[string]webscanner.Result{}}
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
	if _, ok := p.admins[userID]; !ok {
		return true, p.sender.SendMessage(ctx, chatID, denied(language))
	}
	switch cmd {
	case "/start", "/help":
		return true, p.sender.SendMessage(ctx, chatID, help(language))
	case "/status":
		return true, p.sender.SendMessage(ctx, chatID, status(language, len(p.cachedScans())))
	case "/sources":
		return true, p.sender.SendMessage(ctx, chatID, sources(language, p.cachedScans()))
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
	case "/import":
		if len(fields) != 2 {
			return true, p.sender.SendMessage(ctx, chatID, usageImport(language))
		}
		result, ok := p.getScan(fields[1])
		if !ok || result.Status != "succeeded" {
			return true, p.sender.SendMessage(ctx, chatID, unknownScan(language))
		}
		if len(result.Items) > 50 {
			return true, p.sender.SendMessage(ctx, chatID, importFailed(language, "IMPORT_LIMIT"))
		}
		source := "website:" + result.Domain
		provider := "website"
		if len(result.Items) > 0 && strings.HasPrefix(result.Items[0].Source, "greenhouse:") {
			source = result.Items[0].Source
			provider = "greenhouse"
		}
		if err := p.registrar.RegisterDevelopmentSource(ctx, source, provider, result.Domain); err != nil {
			return true, p.sender.SendMessage(ctx, chatID, importFailed(language, "DATABASE_WRITE_FAILED"))
		}
		report, err := ingestion.NewService(p.store, nil, p.logger, 0).Run(ctx, website.Collection{SourceName: source, Items: result.Items, Requests: result.RequestCount, Skipped: result.Skipped}, ingestion.RunOptions{NoAutomaticExpiry: true})
		if err != nil {
			return true, p.sender.SendMessage(ctx, chatID, importFailed(language, report.FailureCategory))
		}
		return true, p.sender.SendMessage(ctx, chatID, importSummary(language, result.ID, report))
	}
	return false, nil
}
func knownCommand(c string) bool {
	switch c {
	case "/start", "/help", "/status", "/scan", "/import", "/sources":
		return true
	}
	return false
}
func (p *CommandProcessor) getScan(id string) (webscanner.Result, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	r, ok := p.scans[id]
	return r, ok
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
		return "Команды:\n/scan <url> — сухое сканирование\n/import <scan_id> — импорт\n/sources — результаты\n/status — состояние"
	}
	return "Командалар:\n/scan <url> — dry-run сканерлеу\n/import <scan_id> — импорт\n/sources — нәтижелер\n/status — күйі"
}
func status(lang string, n int) string {
	if ru(lang) {
		return fmt.Sprintf("Сканер работает. Сохранено результатов: %d", n)
	}
	return fmt.Sprintf("Сканер жұмыс істеп тұр. Сақталған нәтиже: %d", n)
}
func sources(lang string, rows []webscanner.Result) string {
	if len(rows) == 0 {
		if ru(lang) {
			return "Результатов сканирования пока нет."
		}
		return "Сканерлеу нәтижелері әзірге жоқ."
	}
	lines := []string{}
	for _, r := range rows {
		lines = append(lines, fmt.Sprintf("%s — %s (%d)", r.ID, r.Domain, r.Parsed))
	}
	return strings.Join(lines, "\n")
}
func usageScan(lang string) string {
	if ru(lang) {
		return "Использование: /scan https://company.kz"
	}
	return "Қолдану: /scan https://company.kz"
}
func usageImport(lang string) string {
	if ru(lang) {
		return "Использование: /import <scan_id>"
	}
	return "Қолдану: /import <scan_id>"
}
func unknownScan(lang string) string {
	if ru(lang) {
		return "Результат сканирования не найден."
	}
	return "Сканерлеу нәтижесі табылмады."
}
func scanFailed(lang, cat string, n int) string {
	if ru(lang) {
		return fmt.Sprintf("Сканирование не выполнено. Категория: %s\nЗапросов: %d", cat, n)
	}
	return fmt.Sprintf("Сканерлеу орындалмады. Санат: %s\nСұрау: %d", cat, n)
}
func importFailed(lang, cat string) string {
	if ru(lang) {
		return "Импорт не выполнен: " + cat
	}
	return "Импорт орындалмады: " + cat
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
		return fmt.Sprintf("Сканирование завершено\n\nДомен: %s\nCareer: %s\nATS: %s\nЗапросов: %d\nНайдено: %d\nРаспознано: %d\nПропущено: %d\n\n%s\n\nScan ID: %s\nЭто dry-run. В JobHub ничего не записано.\nДля импорта: /import %s", r.Domain, career, ats, r.RequestCount, r.VacancyURLs, r.Parsed, r.Skipped, strings.Join(sample, "\n"), r.ID, r.ID)
	}
	return fmt.Sprintf("Сканерлеу аяқталды\n\nДомен: %s\nCareer: %s\nATS: %s\nСұрау: %d\nТабылды: %d\nТанылды: %d\nӨткізілді: %d\n\n%s\n\nScan ID: %s\nБұл dry-run. JobHub-қа жазылған жоқ.\nИмпорт үшін: /import %s", r.Domain, career, ats, r.RequestCount, r.VacancyURLs, r.Parsed, r.Skipped, strings.Join(sample, "\n"), r.ID, r.ID)
}
func importSummary(lang, id string, r ingestion.Report) string {
	if ru(lang) {
		return fmt.Sprintf("Импорт завершён\nScan ID: %s\nДобавлено: %d\nОбновлено: %d", id, r.Stats.InsertedCount, r.Stats.UpdatedCount)
	}
	return fmt.Sprintf("Импорт аяқталды\nScan ID: %s\nҚосылды: %d\nЖаңартылды: %d", id, r.Stats.InsertedCount, r.Stats.UpdatedCount)
}
