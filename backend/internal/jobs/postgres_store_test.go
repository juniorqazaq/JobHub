package jobs

import "testing"

func TestNormalizedCompanyNamePreventsWhitespaceDuplicates(t *testing.T) {
	if got := normalizedCompanyName("  Kaspi.kz\n\t"); got != "Kaspi.kz" {
		t.Fatalf("normalized company name = %q", got)
	}
}

func TestSourceName(t *testing.T) {
	for source, want := range map[string]string{
		"jobhub":        "JobHub",
		"jooble:kz":     "Jooble",
		"demo:external": "Demo сыртқы дереккөз",
		"partner:kz":    "partner:kz",
	} {
		if got := sourceName(source); got != want {
			t.Errorf("sourceName(%q) = %q, want %q", source, got, want)
		}
	}
}
