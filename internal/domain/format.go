package domain

import "fmt"

type OutputFormat string

const (
	FormatBrief  OutputFormat = "brief"
	FormatJSON   OutputFormat = "json"
	FormatPrompt OutputFormat = "prompt"
)

func ParseOutputFormat(value string) (OutputFormat, error) {
	switch OutputFormat(value) {
	case FormatBrief, FormatJSON, FormatPrompt:
		return OutputFormat(value), nil
	default:
		return "", fmt.Errorf("unsupported format %q (expected brief, json, or prompt)", value)
	}
}
