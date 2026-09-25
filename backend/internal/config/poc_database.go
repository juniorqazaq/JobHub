package config

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strings"
)

var pocDatabaseName = regexp.MustCompile(`^jobhub_[A-Za-z0-9_]*_poc_[A-Za-z0-9_]+$`)

type hostResolver interface {
	LookupIP(context.Context, string, string) ([]net.IP, error)
}
type systemResolver struct{}

func (systemResolver) LookupIP(ctx context.Context, network, host string) ([]net.IP, error) {
	return net.DefaultResolver.LookupIP(ctx, network, host)
}

// ValidateLocalPOCDatabase is the single write guard for local POC databases.
// It intentionally ignores sslmode when deciding locality: the URL host and
// its resolved addresses must independently prove that the target is loopback.
func ValidateLocalPOCDatabase(environment, raw string) error {
	return validateLocalPOCDatabase(context.Background(), environment, raw, systemResolver{})
}

func validateLocalPOCDatabase(ctx context.Context, environment, raw string, resolver hostResolver) error {
	u, parseErr := url.Parse(strings.TrimSpace(raw))
	host, databaseName := "unknown", "unknown"
	if parseErr == nil {
		if u.Hostname() != "" {
			host = u.Hostname()
		}
		if decoded, e := url.PathUnescape(strings.TrimPrefix(u.EscapedPath(), "/")); e == nil && decoded != "" {
			databaseName = decoded
		}
	}
	refused := func(reason string) error {
		return fmt.Errorf("POC database refused: %s (host=%q database=%q)", reason, host, databaseName)
	}
	if parseErr != nil || u.Scheme != "postgres" && u.Scheme != "postgresql" || u.Hostname() == "" || u.Port() == "" || u.Fragment != "" {
		return refused("invalid PostgreSQL connection URL")
	}
	if environment != "development" && environment != "local" {
		return refused("environment must be development or local")
	}
	if !pocDatabaseName.MatchString(databaseName) {
		return refused("database name must match jobhub_*_poc_*")
	}
	hostname := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if ip := net.ParseIP(hostname); ip != nil {
		if !ip.IsLoopback() {
			return refused("database host must be loopback (localhost, 127.0.0.1, or ::1)")
		}
	} else {
		if hostname != "localhost" {
			return refused("database host must be loopback (localhost, 127.0.0.1, or ::1)")
		}
		ips, e := resolver.LookupIP(ctx, "ip", hostname)
		if e != nil || len(ips) == 0 {
			return refused("localhost could not be resolved safely")
		}
		for _, ip := range ips {
			if !ip.IsLoopback() {
				return refused("database host must resolve only to loopback addresses")
			}
		}
	}
	q, e := url.ParseQuery(u.RawQuery)
	if e != nil {
		return refused("invalid URL query")
	}
	for key := range q {
		if key != "sslmode" {
			return refused("database connection overrides are not allowed")
		}
	}
	return nil
}
