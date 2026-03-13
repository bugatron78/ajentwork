package render

import (
	"strings"
	"testing"
	"time"

	"ajentwork/internal/help"
)

func TestManPageIncludesCoreSections(t *testing.T) {
	page := ManPage(help.DefaultRegistry(), time.Date(2026, 3, 13, 0, 0, 0, 0, time.UTC))

	needles := []string{
		".TH AJ 1 \"2026-03-13\" \"ajentwork\" \"Agent Work Tracker\" \"User Commands\"",
		".SH NAME",
		"aj \\- agent work tracker with optional Jira sync",
		".SH COMMANDS",
		".SS JIRA",
		"aj jira <pull|push|link|unlink|status\\-map|transitions|sync|comment> ...",
		".SH WORKFLOWS",
		".SH EXAMPLES",
		".SH CONFIGURATION",
		".SH ENVIRONMENT",
		".SH GLOSSARY",
		".SH FILES",
		".SH SEE ALSO",
	}

	for _, needle := range needles {
		if !strings.Contains(page, needle) {
			t.Fatalf("expected man page to contain %q\n%s", needle, page)
		}
	}
}
