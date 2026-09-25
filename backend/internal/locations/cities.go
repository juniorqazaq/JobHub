package locations

import (
	"strings"
	"unicode"
)

type MatchStatus string

const (
	MatchMatched   MatchStatus = "matched"
	MatchAmbiguous MatchStatus = "ambiguous"
	MatchUnknown   MatchStatus = "unknown"
)

type City struct {
	ID string
	KK string
	RU string
	EN string
}

var Cities = []City{
	{ID: "astana", KK: "Астана", RU: "Астана", EN: "Astana"},
	{ID: "almaty", KK: "Алматы", RU: "Алматы", EN: "Almaty"},
	{ID: "shymkent", KK: "Шымкент", RU: "Шымкент", EN: "Shymkent"},
	{ID: "karaganda", KK: "Қарағанды", RU: "Караганда", EN: "Karaganda"},
	{ID: "aktobe", KK: "Ақтөбе", RU: "Актобе", EN: "Aktobe"},
	{ID: "taraz", KK: "Тараз", RU: "Тараз", EN: "Taraz"},
	{ID: "pavlodar", KK: "Павлодар", RU: "Павлодар", EN: "Pavlodar"},
	{ID: "oskemen", KK: "Өскемен", RU: "Усть-Каменогорск", EN: "Oskemen"},
	{ID: "semey", KK: "Семей", RU: "Семей", EN: "Semey"},
	{ID: "atyrau", KK: "Атырау", RU: "Атырау", EN: "Atyrau"},
	{ID: "kostanay", KK: "Қостанай", RU: "Костанай", EN: "Kostanay"},
	{ID: "kyzylorda", KK: "Қызылорда", RU: "Кызылорда", EN: "Kyzylorda"},
	{ID: "aktau", KK: "Ақтау", RU: "Актау", EN: "Aktau"},
	{ID: "oral", KK: "Орал", RU: "Уральск", EN: "Oral"},
	{ID: "petropavl", KK: "Петропавл", RU: "Петропавловск", EN: "Petropavl"},
	{ID: "turkistan", KK: "Түркістан", RU: "Туркестан", EN: "Turkistan"},
}

var aliases = buildAliases()

func Valid(id string) bool {
	_, ok := aliases[normalize(id)]
	return ok && aliases[normalize(id)] == id
}

func Name(id, language string) string {
	for _, city := range Cities {
		if city.ID != id {
			continue
		}
		switch language {
		case "kk":
			return city.KK
		case "ru":
			return city.RU
		default:
			return city.EN
		}
	}
	return ""
}

// Normalize maps only an exact city alias, optionally followed by the country.
// Regions, free-form places and multi-city locations remain unclassified.
func Normalize(value string) string {
	id, _ := Classify(value)
	return id
}

// Classify distinguishes an unknown location from an explicitly ambiguous
// multi-location value so maintenance tools can report both without guessing.
func Classify(value string) (string, MatchStatus) {
	key := canonicalCandidate(value)
	if id := aliases[key]; id != "" {
		return id, MatchMatched
	}
	if isAmbiguousLocation(value) {
		return "", MatchAmbiguous
	}
	return "", MatchUnknown
}

func buildAliases() map[string]string {
	result := map[string]string{}
	for _, city := range Cities {
		for _, value := range []string{city.ID, city.KK, city.RU, city.EN} {
			result[normalize(value)] = city.ID
		}
	}
	for alias, id := range map[string]string{
		"нур-султан": "astana", "nur-sultan": "astana", "нұр-сұлтан": "astana",
		"ust-kamenogorsk": "oskemen", "ust kamenogorsk": "oskemen", "усть каменогорск": "oskemen",
		"uralsk": "oral", "petropavlovsk": "petropavl",
	} {
		result[normalize(alias)] = id
	}
	return result
}

func normalize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return r
		}
		return ' '
	}, value)
	return strings.Join(strings.Fields(value), " ")
}

func canonicalCandidate(value string) string {
	key := normalize(value)
	for _, prefix := range []string{"г ", "город ", "қ ", "қала ", "city "} {
		key = strings.TrimPrefix(key, prefix)
	}
	for _, country := range []string{"kazakhstan", "казахстан", "қазақстан"} {
		key = strings.TrimSuffix(key, " "+country)
	}
	return strings.TrimSpace(key)
}

func isAmbiguousLocation(value string) bool {
	lower := strings.ToLower(strings.TrimSpace(value))
	for _, separator := range []string{" немесе ", " және ", " или ", " and ", " or ", " / ", "/", ",", ";", "|"} {
		lower = strings.ReplaceAll(lower, separator, "|")
	}
	parts := strings.Split(lower, "|")
	if len(parts) < 2 {
		return false
	}
	meaningful := 0
	hasLocationSignal := false
	for _, part := range parts {
		key := canonicalCandidate(part)
		if key == "" {
			continue
		}
		meaningful++
		if aliases[key] != "" || key == "remote" || key == "удаленно" || key == "қашықтан" {
			hasLocationSignal = true
		}
	}
	return meaningful > 1 && hasLocationSignal
}
