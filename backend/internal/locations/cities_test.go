package locations

import "testing"

func TestNormalizeCityAliases(t *testing.T) {
	tests := map[string]string{
		"Астана": "astana", "Astana": "astana", "г. Астана": "astana", "Нур-Султан": "astana",
		"Алматы": "almaty", "Almaty": "almaty", "г Алматы": "almaty", "Алматы, Казахстан": "almaty",
		"Өскемен": "oskemen", "Ust-Kamenogorsk": "oskemen", "Уральск": "oral",
		"Алматинская область": "", "Astana / Almaty": "", "Almaty, Astana": "", "Astana or remote": "", "Казахстан": "", "": "",
	}
	for input, expected := range tests {
		if actual := Normalize(input); actual != expected {
			t.Errorf("Normalize(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestClassifyDistinguishesAmbiguousAndUnknown(t *testing.T) {
	for _, input := range []string{"Алматы / Астана", "Астана, Алматы", "Astana or remote"} {
		if id, status := Classify(input); id != "" || status != MatchAmbiguous {
			t.Errorf("Classify(%q)=(%q,%q), want ambiguous", input, id, status)
		}
	}
	for _, input := range []string{"Казахстан", "Удаленно", "Алматинская область", ""} {
		if id, status := Classify(input); id != "" || status != MatchUnknown {
			t.Errorf("Classify(%q)=(%q,%q), want unknown", input, id, status)
		}
	}
}

func TestValidUsesCanonicalIDsOnly(t *testing.T) {
	if !Valid("astana") || Valid("Astana") || Valid("nur-sultan") || Valid("unknown") {
		t.Fatal("canonical ID validation is inconsistent")
	}
}
