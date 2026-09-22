package pgtest

import (
	"strings"
	"testing"
)

type fakeTB struct {
	testing.TB
	name string
}

func (f fakeTB) Name() string   { return f.name }
func (f fakeTB) Helper()        {}
func (f fakeTB) Cleanup(func()) {}

func TestDatabaseNameSanitizes(t *testing.T) {
	got := DatabaseName(fakeTB{name: "TestFoo/Bar-2"})
	if got != "t_testfoo_bar_2" {
		t.Fatalf("got %q", got)
	}
}

func TestDatabaseNameTruncates(t *testing.T) {
	long := strings.Repeat("a", 80)
	got := DatabaseName(fakeTB{name: long})
	if len(got) > 63 {
		t.Fatalf("len=%d", len(got))
	}
	if !strings.HasPrefix(got, "t_") {
		t.Fatalf("prefix: %q", got)
	}
}
