package envconfig

import (
	"testing"
	"time"
)

func TestOr(t *testing.T) {
	t.Setenv("ENVCONFIG_OR", "")
	if got := Or("ENVCONFIG_OR", "fb"); got != "fb" {
		t.Fatalf("empty = %q", got)
	}
	t.Setenv("ENVCONFIG_OR", "set")
	if got := Or("ENVCONFIG_OR", "fb"); got != "set" {
		t.Fatalf("set = %q", got)
	}
}

func TestInt(t *testing.T) {
	t.Setenv("ENVCONFIG_INT", "")
	n, err := Int("ENVCONFIG_INT", 7)
	if err != nil || n != 7 {
		t.Fatalf("empty: n=%d err=%v", n, err)
	}
	t.Setenv("ENVCONFIG_INT", "9")
	n, err = Int("ENVCONFIG_INT", 7)
	if err != nil || n != 9 {
		t.Fatalf("set: n=%d err=%v", n, err)
	}
	t.Setenv("ENVCONFIG_INT", "nope")
	if _, err = Int("ENVCONFIG_INT", 7); err == nil {
		t.Fatal("expected error")
	}
}

func TestInt64(t *testing.T) {
	t.Setenv("ENVCONFIG_I64", "42")
	n, err := Int64("ENVCONFIG_I64", 1)
	if err != nil || n != 42 {
		t.Fatalf("n=%d err=%v", n, err)
	}
}

func TestDuration(t *testing.T) {
	t.Setenv("ENVCONFIG_DUR", "")
	d, err := Duration("ENVCONFIG_DUR", time.Minute)
	if err != nil || d != time.Minute {
		t.Fatalf("empty: %v %v", d, err)
	}
	t.Setenv("ENVCONFIG_DUR", "2s")
	d, err = Duration("ENVCONFIG_DUR", time.Minute)
	if err != nil || d != 2*time.Second {
		t.Fatalf("set: %v %v", d, err)
	}
	t.Setenv("ENVCONFIG_DUR", "bogus")
	if _, err = Duration("ENVCONFIG_DUR", time.Minute); err == nil {
		t.Fatal("expected error")
	}
}

func TestBool(t *testing.T) {
	t.Setenv("ENVCONFIG_BOOL", "false")
	b, err := Bool("ENVCONFIG_BOOL", true)
	if err != nil || b {
		t.Fatalf("b=%v err=%v", b, err)
	}
}

func TestTruthy(t *testing.T) {
	t.Setenv("ENVCONFIG_TRUTHY", "YES")
	if !Truthy("ENVCONFIG_TRUTHY") {
		t.Fatal("expected true")
	}
	t.Setenv("ENVCONFIG_TRUTHY", "no")
	if Truthy("ENVCONFIG_TRUTHY") {
		t.Fatal("expected false")
	}
}

func TestFirstNonEmpty(t *testing.T) {
	if got := FirstNonEmpty("", "  ", "a", "b"); got != "a" {
		t.Fatalf("got=%q", got)
	}
	if got := FirstNonEmpty("", " "); got != "" {
		t.Fatalf("got=%q", got)
	}
}
