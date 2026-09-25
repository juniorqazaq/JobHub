package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"regexp"
	"strings"
)

type ATSBoard struct {
	BoardToken       string `json:"board_token"`
	DisplayName      string `json:"display_name"`
	AuthorizedForPOC bool   `json:"authorized_for_poc"`
	TestBoard        bool   `json:"test_board"`
}
type ATSConfig struct {
	Boards        []ATSBoard     `json:"boards"`
	CareerSources []CareerSource `json:"career_sources"`
	MaxRequests   int            `json:"max_requests"`
	MaxJobs       int            `json:"max_jobs"`
}
type CareerSource struct {
	Provider         string `json:"provider"`
	DisplayName      string `json:"display_name"`
	AuthorizedForPOC bool   `json:"authorized_for_poc"`
}

func LoadATS(path string) (ATSConfig, error) {
	var c ATSConfig
	f, err := os.Open(path)
	if err != nil {
		return c, errors.New("cannot read ATS config")
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, 65537))
	if err != nil || len(raw) > 65536 {
		return c, errors.New("invalid ATS config size")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&c) != nil || decoder.Decode(new(any)) != io.EOF {
		return c, errors.New("invalid ATS config JSON")
	}
	if c.MaxRequests == 0 {
		c.MaxRequests = 12
	}
	if c.MaxJobs == 0 {
		c.MaxJobs = 200
	}
	if c.MaxRequests < 1 || c.MaxRequests > 12 || c.MaxJobs < 1 || c.MaxJobs > 200 || len(c.Boards) > 3 || len(c.Boards)+len(c.CareerSources) < 1 || len(c.CareerSources) > 2 {
		return c, errors.New("ATS POC limits exceeded")
	}
	seen := map[string]bool{}
	normal, test := 0, 0
	for _, b := range c.Boards {
		if !regexp.MustCompile(`^[A-Za-z0-9_-]{1,100}$`).MatchString(b.BoardToken) || strings.TrimSpace(b.DisplayName) == "" || len(b.DisplayName) > 200 || seen[b.BoardToken] {
			return c, errors.New("invalid or duplicate ATS board")
		}
		seen[b.BoardToken] = true
		if b.TestBoard {
			test++
		} else {
			normal++
		}
	}
	if normal > 2 || test > 1 {
		return c, errors.New("at most two boards and one test board are permitted")
	}
	for _, s := range c.CareerSources {
		if (s.Provider != "kcell" && s.Provider != "airastana") || strings.TrimSpace(s.DisplayName) == "" || seen[s.Provider+":careers"] {
			return c, errors.New("invalid or duplicate career source")
		}
		seen[s.Provider+":careers"] = true
	}
	return c, nil
}

// ValidateATSDevelopmentDatabase intentionally accepts only explicit loopback
// databases dedicated to this POC. Query parameters cannot override host/service.
func ValidateATSDevelopmentDatabase(environment, raw string) error {
	return ValidateLocalPOCDatabase(environment, raw)
}
