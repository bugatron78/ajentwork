package render

import (
	"fmt"
	"strings"

	"ajentwork/internal/store"
)

func JiraSpaceExistsBrief(result store.JiraSpaceExistsResult) string {
	lines := []string{
		fmt.Sprintf("Key: %s", fallbackValue(result.Key)),
		fmt.Sprintf("Exists: %t", result.Exists),
	}
	if result.Exists {
		lines = append(lines,
			fmt.Sprintf("Name: %s", fallbackValue(result.Space.Name)),
			fmt.Sprintf("Type: %s", fallbackValue(result.Space.ProjectTypeKey)),
		)
	}
	return strings.Join(lines, "\n")
}

func JiraSpaceExistsPrompt(result store.JiraSpaceExistsResult) string {
	lines := []string{
		"Key: " + fallbackValue(result.Key),
		fmt.Sprintf("Exists: %t", result.Exists),
	}
	if result.Exists {
		lines = append(lines,
			"Name: "+fallbackValue(result.Space.Name),
			"Type: "+fallbackValue(result.Space.ProjectTypeKey),
		)
	}
	return strings.Join(lines, "\n")
}

func JiraSpaceCreateBrief(result store.JiraSpaceCreateResult) string {
	if result.Created {
		return fmt.Sprintf("created Jira space %s (%s)", result.Space.Key, result.Space.Name)
	}
	return fmt.Sprintf("using existing Jira space %s (%s)", result.Space.Key, result.Space.Name)
}

func JiraSpaceCreatePrompt(result store.JiraSpaceCreateResult) string {
	status := "existing Jira space"
	if result.Created {
		status = "created Jira space"
	}
	return strings.Join([]string{
		"Status: " + status,
		"Key: " + result.Space.Key,
		"Name: " + result.Space.Name,
		"Type: " + fallbackValue(result.Space.ProjectTypeKey),
	}, "\n")
}

func JiraSpaceListBrief(result store.JiraSpaceListResult) string {
	lines := []string{
		fmt.Sprintf("Query: %s", fallbackValue(result.Query)),
		fmt.Sprintf("Matches: %d", len(result.Spaces)),
	}
	if len(result.Spaces) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Spaces:")
	for _, space := range result.Spaces {
		lines = append(lines, fmt.Sprintf("  %-10s %-10s %s", space.Key, fallbackValue(space.ProjectTypeKey), fallbackValue(space.Name)))
	}
	return strings.Join(lines, "\n")
}

func JiraSpaceListPrompt(result store.JiraSpaceListResult) string {
	lines := []string{
		"Query: " + fallbackValue(result.Query),
		fmt.Sprintf("Matches: %d", len(result.Spaces)),
	}
	for _, space := range result.Spaces {
		lines = append(lines, fmt.Sprintf("%s type=%s name=%s", space.Key, fallbackValue(space.ProjectTypeKey), fallbackValue(space.Name)))
	}
	return strings.Join(lines, "\n")
}
