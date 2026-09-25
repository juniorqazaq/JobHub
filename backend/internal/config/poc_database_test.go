package config

import (
	"context"
	"net"
	"strings"
	"testing"
)

type resolverFunc func(context.Context, string, string) ([]net.IP, error)

func (f resolverFunc) LookupIP(c context.Context, n, h string) ([]net.IP, error) { return f(c, n, h) }

func TestLocalPOCDatabaseGuard(t *testing.T) {
	loopback := resolverFunc(func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}, nil
	})
	allowed := []string{"postgres://user:password@localhost:5432/jobhub_test_poc_one?sslmode=disable", "postgres://user:password@127.0.0.1:5432/jobhub_test_poc_two", "postgres://user:password@[::1]:5432/jobhub_test_poc_three"}
	for _, raw := range allowed {
		if err := validateLocalPOCDatabase(context.Background(), "development", raw, loopback); err != nil {
			t.Errorf("rejected %s: %v", raw, err)
		}
	}
	rejected := []string{"postgres://u:SECRET@db.supabase.co:5432/jobhub_test_poc_x", "postgres://u:SECRET@aws.pooler.supabase.com:6543/jobhub_test_poc_x", "postgres://u:SECRET@8.8.8.8:5432/jobhub_test_poc_x", "postgres://u:SECRET@192.168.1.2:5432/jobhub_test_poc_x", "postgres://u:SECRET@10.0.0.2:5432/jobhub_test_poc_x", "postgres://u:SECRET@172.16.0.2:5432/jobhub_test_poc_x", "postgres://u:SECRET@localhost:5432/production"}
	for _, raw := range rejected {
		err := validateLocalPOCDatabase(context.Background(), "development", raw, loopback)
		if err == nil {
			t.Errorf("accepted %s", raw)
		} else if strings.Contains(err.Error(), "SECRET") {
			t.Fatalf("password leaked: %v", err)
		}
	}
	for _, env := range []string{"", "test", "staging", "production"} {
		if validateLocalPOCDatabase(context.Background(), env, allowed[0], loopback) == nil {
			t.Errorf("accepted environment %q", env)
		}
	}
	mixed := resolverFunc(func(context.Context, string, string) ([]net.IP, error) {
		return []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.1.2")}, nil
	})
	if validateLocalPOCDatabase(context.Background(), "local", allowed[0], mixed) == nil {
		t.Fatal("accepted localhost with non-loopback DNS result")
	}
}
