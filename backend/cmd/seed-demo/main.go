package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"
	"jobhub-ai/backend/internal/config"
	"jobhub-ai/backend/internal/database"
)

type demoCompany struct {
	Name, Description, Website, Logo, Industry, City string
	Verified                                         bool
}

type demoVacancy struct {
	Company, ExternalID, Title, City, Description string
	External                                      bool
}

var demoCompanies = []demoCompany{
	{"Air Astana", "Қазақстанның әуе тасымалдаушысы.", "https://airastana.com", "/company-logos/air-astana.svg", "Авиация", "Алматы", true},
	{"Kaspi.kz", "Қазақстандағы технологиялық және қаржылық экожүйе.", "https://kaspi.kz", "/company-logos/kaspi.png", "Технологиялар және қаржы", "Алматы", true},
	{"Halyk Bank", "Қазақстандағы банк және қаржылық қызметтер тобы.", "https://halykbank.kz", "/company-logos/halyk-bank.svg", "Банктік қызметтер", "Алматы", true},
	{"Kcell", "Қазақстандық телекоммуникация операторы.", "https://kcell.kz", "/company-logos/kcell.svg", "Телекоммуникация", "Алматы", false},
	{"ForteBank", "Қазақстандағы әмбебап коммерциялық банк.", "https://forte.kz", "/company-logos/forte.ico", "Банктік қызметтер", "Алматы", false},
	{"Kolesa Group", "Қазақстандық classifieds және mobility технологиялар тобы.", "https://kolesa.group", "/company-logos/kolesa-group.svg", "Технологиялар", "Алматы", false},
	{"Technodom", "Электроника және тұрмыстық техника ритейлері.", "https://www.technodom.kz", "/company-logos/technodom.svg", "Ритейл", "Алматы", false},
	{"BI Group", "Құрылыс және девелопмент компаниясы.", "https://bi.group", "", "Құрылыс", "Астана", false},
	{"Magnum", "Қазақстандық азық-түлік ритейл желісі.", "https://magnum.kz", "/company-logos/magnum.ico", "Ритейл", "Алматы", false},
	{"Freedom Bank", "Цифрлық банк және қаржылық сервис.", "https://bankffin.kz", "/company-logos/freedom.svg", "Қаржылық технологиялар", "Алматы", false},
}

var demoVacancies = []demoVacancy{
	{"Kaspi.kz", "", "Product Manager", "Алматы", "Компания профиліне арналған JobHub demo вакансиясы.", false},
	{"Air Astana", "", "Data Analyst", "Алматы", "Компания профиліне арналған JobHub demo вакансиясы.", false},
	{"Kcell", "demo-kcell-network-engineer", "Network Engineer", "Алматы", "Demo импортталған сыртқы вакансия. Өтінішті сыртқы дереккөзде беріңіз.", true},
	{"Kolesa Group", "demo-kolesa-go-engineer", "Go Engineer", "Алматы", "Demo импортталған сыртқы вакансия. Өтінішті сыртқы дереккөзде беріңіз.", true},
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	if err := run(context.Background()); err != nil {
		logger.Error("demo seed failed", "error", err)
		os.Exit(1)
	}
	logger.Info("demo data seeded", "companies", len(demoCompanies), "vacancies", len(demoVacancies))
}

func run(ctx context.Context) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if err := validateSeedTarget(cfg.Environment, cfg.DatabaseURL, os.Getenv("JOBHUB_ALLOW_REMOTE_DEMO_SEED")); err != nil {
		return err
	}
	pool, err := database.Connect(ctx, cfg.DatabaseURL, cfg.PGXQueryExecMode)
	if err != nil {
		return err
	}
	defer pool.Close()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `INSERT INTO jobhub.job_sources (source, provider, market, display_name, enabled, production_permissions_confirmed)
		VALUES ('demo:external', 'demo', 'KZ', 'Demo сыртқы дереккөз', true, true)
		ON CONFLICT (source) DO UPDATE SET display_name = EXCLUDED.display_name, enabled = true, production_permissions_confirmed = true`); err != nil {
		return fmt.Errorf("create demo source: %w", err)
	}
	var ownerID pgtype.UUID
	if err := tx.QueryRow(ctx, `INSERT INTO jobhub.users (full_name, email, normalized_email, password_hash, role, status)
		VALUES ('Demo employer', 'demo-employer@jobhub.local', 'demo-employer@jobhub.local', 'demo-only-not-for-login', 'employer', 'active')
		ON CONFLICT (normalized_email) DO UPDATE SET status = 'active'
		RETURNING id`).Scan(&ownerID); err != nil {
		return fmt.Errorf("create demo owner: %w", err)
	}
	companyIDs := make(map[string]pgtype.UUID, len(demoCompanies))
	for _, company := range demoCompanies {
		var id pgtype.UUID
		err := tx.QueryRow(ctx, `INSERT INTO jobhub.companies (name, description, website_url, logo_url, industry, city, is_verified)
			VALUES ($1,$2,$3,NULLIF($4,''),$5,$6,$7)
			ON CONFLICT (lower(regexp_replace(btrim(name), '\s+', ' ', 'g')))
			DO UPDATE SET description=EXCLUDED.description, website_url=EXCLUDED.website_url, logo_url=EXCLUDED.logo_url,
			industry=EXCLUDED.industry, city=EXCLUDED.city, is_verified=EXCLUDED.is_verified, updated_at=now()
			RETURNING id`, company.Name, company.Description, company.Website, company.Logo, company.Industry, company.City, company.Verified).Scan(&id)
		if err != nil {
			return fmt.Errorf("upsert demo company %q: %w", company.Name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO jobhub.company_memberships (company_id, user_id, role)
			VALUES ($1,$2,'owner') ON CONFLICT (company_id, user_id) DO NOTHING`, id, ownerID); err != nil {
			return fmt.Errorf("assign demo owner to %q: %w", company.Name, err)
		}
		companyIDs[company.Name] = id
	}
	for _, vacancy := range demoVacancies {
		companyID := companyIDs[vacancy.Company]
		cityID := "almaty"
		if vacancy.External {
			_, err = tx.Exec(ctx, `INSERT INTO jobhub.jobs (source, external_id, source_url, company_id, company_name_raw, title, location_raw, canonical_city_id, description, description_kind, application_method, apply_url, source_status, moderation_status, first_seen_at, last_seen_at, last_synced_at, fresh_until, external_published_at)
				VALUES ('demo:external',$1,'https://demo.invalid/vacancies/' || $1,$2,$3,$4,$5,$6,$7,'full','external','https://demo.invalid/vacancies/' || $1,'active','approved',now(),now(),now(),now() + interval '30 days',now())
				ON CONFLICT (source, external_id) DO UPDATE SET company_id=EXCLUDED.company_id, company_name_raw=EXCLUDED.company_name_raw, title=EXCLUDED.title, description=EXCLUDED.description, updated_at=now()`,
				vacancy.ExternalID, companyID, vacancy.Company, vacancy.Title, vacancy.City, cityID, vacancy.Description)
		} else {
			_, err = tx.Exec(ctx, `INSERT INTO jobhub.jobs (source, company_id, created_by_user_id, company_name_raw, title, location_raw, canonical_city_id, description, description_kind, application_method, source_status, publication_status, moderation_status, first_seen_at, last_seen_at, last_synced_at, fresh_until, published_at)
				SELECT 'jobhub',$1,$2,$3,$4,$5,$6,$7,'full','internal','active','published','approved',now(),now(),now(),'infinity'::timestamptz,now()
				WHERE NOT EXISTS (SELECT 1 FROM jobhub.jobs WHERE source='jobhub' AND company_id=$1 AND title=$4)`,
				companyID, ownerID, vacancy.Company, vacancy.Title, vacancy.City, cityID, vacancy.Description)
			if err == nil {
				_, err = tx.Exec(ctx, `UPDATE jobhub.jobs SET description=$1, updated_at=now()
					WHERE source='jobhub' AND company_id=$2 AND title=$3`, vacancy.Description, companyID, vacancy.Title)
			}
		}
		if err != nil {
			return fmt.Errorf("upsert demo vacancy %q: %w", vacancy.Title, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	return nil
}

// validateSeedTarget keeps demo fixtures local by default. A staging run needs
// an explicit, per-command opt-in so sourcing .env cannot seed a remote target
// accidentally. Production remains prohibited.
func validateSeedTarget(environment, databaseURL, remoteOptIn string) error {
	if err := config.ValidateLocalPOCDatabase(environment, databaseURL); err == nil {
		return nil
	} else if environment == "staging" && remoteOptIn == "true" {
		return nil
	} else {
		return err
	}
}

func validateDemoData() error {
	if len(demoCompanies) != 10 || len(demoVacancies) < 4 {
		return errors.New("demo catalog is incomplete")
	}
	seen := map[string]bool{}
	verifiedWithoutJobs := false
	for _, company := range demoCompanies {
		if company.Name == "" || seen[company.Name] {
			return errors.New("demo company names must be unique")
		}
		seen[company.Name] = true
		if company.Verified && company.Name == "Halyk Bank" {
			verifiedWithoutJobs = true
		}
	}
	if !verifiedWithoutJobs || !strings.Contains(demoVacancies[2].ExternalID, "demo-") {
		return errors.New("demo vacancy coverage is incomplete")
	}
	return nil
}
