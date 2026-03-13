package render

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ajentwork/internal/help"
)

func ManPage(registry help.Registry, generatedAt time.Time) string {
	root := registry.Root()
	commands := registry.Commands()
	workflows := registry.Workflows()
	examples := registry.ExampleSets()
	glossary := registry.Glossary()

	var lines []string
	lines = append(lines, fmt.Sprintf(".TH AJ 1 %q %q %q %q", generatedAt.UTC().Format("2006-01-02"), "ajentwork", "Agent Work Tracker", "User Commands"))
	lines = append(lines, ".SH NAME")
	lines = append(lines, roffEscape("aj - "+root.Summary))
	lines = append(lines, ".SH SYNOPSIS")
	lines = append(lines, ".B aj")
	lines = append(lines, roffEscape("<command> [options]"))
	lines = append(lines, ".SH DESCRIPTION")
	lines = append(lines, roffEscape(root.Purpose))
	lines = append(lines, roffEscape("The primary interactive help surfaces remain `aj --help`, `aj help <command>`, `aj workflows`, and `aj examples`. This man page is a generated offline companion."))
	lines = append(lines, ".SH COMMANDS")
	for _, command := range commands {
		doc, ok := registry.Command(command.Name)
		if !ok {
			continue
		}
		lines = append(lines, ".SS "+strings.ToUpper(roffEscape(command.Name)))
		lines = append(lines, ".B "+roffEscape(doc.Usage))
		lines = append(lines, roffEscape(doc.Purpose))
		if len(doc.Options) > 0 {
			lines = append(lines, ".RS")
			for _, option := range doc.Options {
				lines = append(lines, ".TP")
				lines = append(lines, ".B "+roffEscape(option.Name))
				lines = append(lines, roffEscape(option.Description))
			}
			lines = append(lines, ".RE")
		}
	}
	lines = append(lines, ".SH WORKFLOWS")
	for _, workflow := range workflows {
		lines = append(lines, ".SS "+strings.ToUpper(roffEscape(workflow.Name)))
		lines = append(lines, roffEscape(workflow.Topic))
		for _, step := range workflow.Steps {
			lines = append(lines, ".IP \\[bu] 2")
			lines = append(lines, roffEscape(step))
		}
	}
	lines = append(lines, ".SH EXAMPLES")
	for _, set := range examples {
		lines = append(lines, ".SS "+strings.ToUpper(roffEscape(set.Topic)))
		for _, example := range set.Examples {
			lines = append(lines, ".IP \\[bu] 2")
			lines = append(lines, roffEscape(example.Label+": "+example.Command))
		}
	}
	lines = append(lines, ".SH CONFIGURATION")
	lines = append(lines, roffEscape("aj stores repo-local configuration in .aj/config.toml. Common Jira settings include [jira], [jira.status_map], and [jira.lifecycle]."))
	lines = append(lines, ".nf")
	lines = append(lines, roffEscape("schema_version = 1"))
	lines = append(lines, roffEscape("default_output = \"brief\""))
	lines = append(lines, roffEscape("default_lease_ttl = \"4h\""))
	lines = append(lines, roffEscape(""))
	lines = append(lines, roffEscape("[jira]"))
	lines = append(lines, roffEscape("enabled = true"))
	lines = append(lines, roffEscape("base_url = \"https://example.atlassian.net\""))
	lines = append(lines, roffEscape("project = \"ABC\""))
	lines = append(lines, roffEscape(""))
	lines = append(lines, roffEscape("[jira.lifecycle]"))
	lines = append(lines, roffEscape("comment_on_done = false"))
	lines = append(lines, roffEscape("comment_on_block = false"))
	lines = append(lines, roffEscape("comment_on_handoff = false"))
	lines = append(lines, ".fi")
	lines = append(lines, ".SH ENVIRONMENT")
	envLines := []string{
		"AJ_JIRA_EMAIL - Jira Cloud email used for authenticated Jira commands.",
		"AJ_JIRA_API_TOKEN - Jira Cloud API token used for authenticated Jira commands.",
		"AJ_JIRA_ENABLED - optional override for [jira].enabled.",
		"AJ_JIRA_BASE_URL - optional override for [jira].base_url.",
		"AJ_JIRA_PROJECT - optional override for [jira].project.",
	}
	for _, envLine := range envLines {
		lines = append(lines, ".IP \\[bu] 2")
		lines = append(lines, roffEscape(envLine))
	}
	lines = append(lines, ".SH GLOSSARY")
	for _, entry := range glossary {
		lines = append(lines, ".TP")
		lines = append(lines, ".B "+roffEscape(entry.Term))
		lines = append(lines, roffEscape(entry.Definition))
	}
	lines = append(lines, ".SH FILES")
	files := []string{
		".aj/config.toml - repo-local aj configuration.",
		".aj/issues/ - compact work item snapshots and append-only events.",
		".aj/artifacts/ - large logs or supporting outputs referenced by items.",
	}
	for _, fileLine := range files {
		lines = append(lines, ".IP \\[bu] 2")
		lines = append(lines, roffEscape(fileLine))
	}
	lines = append(lines, ".SH SEE ALSO")
	related := []string{"aj --help", "aj help jira", "aj workflows", "aj examples", "aj glossary"}
	sort.Strings(related)
	lines = append(lines, roffEscape(strings.Join(related, ", ")))
	lines = append(lines, "")
	return strings.Join(lines, "\n")
}

func roffEscape(value string) string {
	value = strings.ReplaceAll(value, `\`, `\\`)
	value = strings.ReplaceAll(value, "-", `\-`)
	if strings.HasPrefix(value, ".") || strings.HasPrefix(value, "'") {
		value = `\&` + value
	}
	return value
}
