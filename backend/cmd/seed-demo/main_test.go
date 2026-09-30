package main

import "testing"

func TestValidateSeedTarget(t *testing.T) {
	tests := []struct {
		name, environment, databaseURL, remoteOptIn string
		wantErr                                     bool
	}{
		{"local POC", "development", "postgresql://localhost:5432/jobhub_companies_poc_20260930?sslmode=disable", "", false},
		{"staging requires explicit opt in", "staging", "postgresql://db.example.com:5432/postgres?sslmode=require", "", true},
		{"staging opt in", "staging", "postgresql://db.example.com:5432/postgres?sslmode=require", "true", false},
		{"production prohibited", "production", "postgresql://db.example.com:5432/postgres?sslmode=require", "true", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateSeedTarget(tt.environment, tt.databaseURL, tt.remoteOptIn)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateSeedTarget() error = %v, wantErr %t", err, tt.wantErr)
			}
		})
	}
}

func TestDemoDataCoversCompanyFlow(t *testing.T) {
	if err := validateDemoData(); err != nil {
		t.Fatal(err)
	}
	var native, external int
	for _, vacancy := range demoVacancies {
		if vacancy.External {
			external++
		} else {
			native++
		}
	}
	if native < 2 || external < 2 {
		t.Fatalf("want at least two native and external vacancies, got native=%d external=%d", native, external)
	}
	for _, name := range []string{"Air Astana", "Kaspi.kz", "Halyk Bank", "Kcell", "ForteBank", "Kolesa Group", "Technodom", "BI Group", "Magnum", "Freedom Bank"} {
		found := false
		for _, company := range demoCompanies {
			found = found || company.Name == name
		}
		if !found {
			t.Fatalf("missing curated demo company %q", name)
		}
	}
	for _, vacancy := range demoVacancies {
		if vacancy.Company == "Halyk Bank" {
			t.Fatal("Halyk Bank must remain a verified empty-state fixture")
		}
	}
}
