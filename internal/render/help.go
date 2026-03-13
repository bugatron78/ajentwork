package render

import (
	"encoding/json"
	"fmt"
	"strings"

	"ajentwork/internal/help"
)

func RootHelp(doc help.Doc, commands []help.CommandSummary) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("aj: %s", doc.Summary))
	lines = append(lines, "")
	lines = append(lines, "Usage:")
	lines = append(lines, "  "+doc.Usage)
	lines = append(lines, "")
	lines = append(lines, "Core:")
	for _, cmd := range commands {
		if cmd.Group != "Core" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-10s %s", cmd.Name, cmd.Summary))
	}
	lines = append(lines, "")
	lines = append(lines, "Discovery:")
	for _, cmd := range commands {
		if cmd.Group != "Discovery" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %-10s %s", cmd.Name, cmd.Summary))
	}
	return strings.Join(lines, "\n")
}

func CommandHelpBrief(doc help.Doc) string {
	var lines []string
	lines = append(lines, doc.Summary)
	lines = append(lines, "")
	lines = append(lines, "Purpose:")
	lines = append(lines, "  "+doc.Purpose)
	lines = append(lines, "")
	lines = append(lines, "Usage:")
	lines = append(lines, "  "+doc.Usage)
	if len(doc.Arguments) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Arguments:")
		for _, arg := range doc.Arguments {
			required := "optional"
			if arg.Required {
				required = "required"
			}
			lines = append(lines, fmt.Sprintf("  %-16s %s (%s)", arg.Name, arg.Description, required))
		}
	}
	if len(doc.Options) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Options:")
		for _, opt := range doc.Options {
			lines = append(lines, fmt.Sprintf("  %-16s %s", opt.Name, opt.Description))
		}
	}
	if len(doc.Examples) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Examples:")
		for _, example := range doc.Examples {
			lines = append(lines, fmt.Sprintf("  %-24s %s", example.Label+":", example.Command))
		}
	}
	if len(doc.Related) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Related:")
		lines = append(lines, "  "+strings.Join(doc.Related, ", "))
	}
	if len(doc.Safety) > 0 {
		lines = append(lines, "")
		lines = append(lines, "Safety:")
		for _, note := range doc.Safety {
			lines = append(lines, "  "+note)
		}
	}
	return strings.Join(lines, "\n")
}

func CommandHelpPrompt(doc help.Doc) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("Command: aj %s", doc.Name))
	lines = append(lines, "Purpose: "+doc.Purpose)
	lines = append(lines, "Usage: "+doc.Usage)
	if len(doc.Arguments) > 0 {
		var args []string
		for _, arg := range doc.Arguments {
			args = append(args, arg.Name)
		}
		lines = append(lines, "Arguments: "+strings.Join(args, ", "))
	}
	if len(doc.Options) > 0 {
		var opts []string
		for _, opt := range doc.Options {
			opts = append(opts, opt.Name)
		}
		lines = append(lines, "Options: "+strings.Join(opts, ", "))
	}
	if len(doc.Related) > 0 {
		lines = append(lines, "Related: "+strings.Join(doc.Related, ", "))
	}
	return strings.Join(lines, "\n")
}

func JSON(value any) (string, error) {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

func CommandsBrief(commands []help.CommandSummary) string {
	var lines []string
	currentGroup := ""
	for _, cmd := range commands {
		if cmd.Group != currentGroup {
			if len(lines) > 0 {
				lines = append(lines, "")
			}
			currentGroup = cmd.Group
			lines = append(lines, currentGroup+":")
		}
		lines = append(lines, fmt.Sprintf("  %-10s %s", cmd.Name, cmd.Summary))
	}
	return strings.Join(lines, "\n")
}

func WorkflowsBrief(workflows []help.WorkflowDoc) string {
	var lines []string
	for i, workflow := range workflows {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, fmt.Sprintf("%s: %s", workflow.Name, workflow.Topic))
		for _, step := range workflow.Steps {
			lines = append(lines, "  "+step)
		}
	}
	return strings.Join(lines, "\n")
}

func WorkflowPrompt(workflow help.WorkflowDoc) string {
	return strings.Join(append([]string{fmt.Sprintf("Workflow: %s", workflow.Name), "Topic: " + workflow.Topic}, workflow.Steps...), "\n")
}

func ExampleSetsBrief(exampleSets []help.ExampleSet) string {
	var lines []string
	for i, set := range exampleSets {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, set.Topic+":")
		for _, example := range set.Examples {
			lines = append(lines, fmt.Sprintf("  %-24s %s", example.Label+":", example.Command))
		}
	}
	return strings.Join(lines, "\n")
}

func GlossaryBrief(entries []help.GlossaryEntry) string {
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%-12s %s", entry.Term, entry.Definition))
	}
	return strings.Join(lines, "\n")
}

func SearchBrief(results []help.SearchResult) string {
	if len(results) == 0 {
		return "No help topics matched your search."
	}
	var lines []string
	for _, result := range results {
		lines = append(lines, fmt.Sprintf("%-10s %-12s %s", result.Kind, result.Name, result.Summary))
	}
	return strings.Join(lines, "\n")
}
