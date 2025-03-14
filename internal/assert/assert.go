package assert

import (
	"strings"
	"testing"
)

func Equal[T comparable](t *testing.T, acutal, expected T) {
	t.Helper()

	if acutal != expected {
		t.Errorf("got: %v; want %v", acutal, expected)
	}
}

func StringContains(t *testing.T, actual, expectedSubstring string) {
	t.Helper()

	if !strings.Contains(actual, expectedSubstring) {
		t.Errorf("got: %q; expected to contain: %q", actual, expectedSubstring)
	}
}
