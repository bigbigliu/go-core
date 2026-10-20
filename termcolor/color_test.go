package termcolor

import (
	"strings"
	"testing"
)

func TestColorDisabledAndEnabled(t *testing.T) {
	c := NewColor()

	c.Disable()
	if got := c.Red("hello"); got != "hello" {
		t.Errorf("disabled Red() = %q, want %q", got, "hello")
	}

	c.Enable()
	if got := c.Red("hello"); !strings.Contains(got, "\x1b[31m") {
		t.Errorf("enabled Red() = %q, want ANSI escape", got)
	}
}
