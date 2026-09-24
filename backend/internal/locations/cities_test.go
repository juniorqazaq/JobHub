package locations

import "testing"

func TestNormalizeCityAliases(t *testing.T) {
	tests := map[string]string{
		"Astana": "astana", "Нур-Султан": "astana", "Алматы, Казахстан": "almaty",
		"Өскемен": "oskemen", "Ust-Kamenogorsk": "oskemen", "Уральск": "oral",
		"Алматинская область": "", "Astana / Almaty": "", "Almaty, Astana": "", "": "",
	}
	for input, expected := range tests {
		if actual := Normalize(input); actual != expected {
			t.Errorf("Normalize(%q)=%q, want %q", input, actual, expected)
		}
	}
}

func TestValidUsesCanonicalIDsOnly(t *testing.T) {
	if !Valid("astana") || Valid("Astana") || Valid("nur-sultan") || Valid("unknown") {
		t.Fatal("canonical ID validation is inconsistent")
	}
}
