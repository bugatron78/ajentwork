package render

import (
	"fmt"
	"strings"

	"ajentwork/internal/domain"
	"ajentwork/internal/store"
)

func ItemCreatedBrief(item domain.Item) string {
	return fmt.Sprintf("created %s [%s] %s", item.ID, item.Kind, item.Title)
}

func ItemCreatedPrompt(item domain.Item) string {
	return strings.Join([]string{
		"Status: created",
		"ID: " + item.ID,
		"Kind: " + string(item.Kind),
		"Title: " + item.Title,
		"Next: " + item.NextAction,
	}, "\n")
}

func ItemListBrief(items []domain.Item) string {
	if len(items) == 0 {
		return "No work items found."
	}
	lines := make([]string, 0, len(items))
	for _, item := range items {
		lease := "-"
		if item.Lease != nil {
			lease = "@" + item.Lease.Owner
		}
		deps := "-"
		if len(item.DependsOn) > 0 {
			deps = fmt.Sprintf("deps=%d", len(item.DependsOn))
		}
		lines = append(lines, fmt.Sprintf("%-12s %-12s p%d %-8s %-12s %-8s %s", item.ID, item.Status, item.Priority, item.Kind, lease, deps, item.Title))
	}
	return strings.Join(lines, "\n")
}

func ItemListPrompt(items []domain.Item) string {
	if len(items) == 0 {
		return "Items: none"
	}
	lines := make([]string, 0, len(items)+1)
	lines = append(lines, fmt.Sprintf("Items: %d", len(items)))
	for _, item := range items {
		owner := "-"
		if item.Lease != nil {
			owner = item.Lease.Owner
		}
		lines = append(lines, fmt.Sprintf("%s %s owner=%s deps=%d %s", item.ID, item.Status, owner, len(item.DependsOn), item.Title))
	}
	return strings.Join(lines, "\n")
}

func ItemShowBrief(item domain.Item) string {
	lines := []string{
		fmt.Sprintf("ID: %s", item.ID),
		fmt.Sprintf("Kind: %s", item.Kind),
		fmt.Sprintf("Title: %s", item.Title),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Priority: %d", item.Priority),
		fmt.Sprintf("Created: %s", item.CreatedAt.Format("2006-01-02T15:04:05Z")),
		fmt.Sprintf("Updated: %s", item.UpdatedAt.Format("2006-01-02T15:04:05Z")),
		fmt.Sprintf("Goal: %s", item.Goal),
		fmt.Sprintf("Summary: %s", item.Summary),
		fmt.Sprintf("Next: %s", item.NextAction),
	}
	if item.Lease != nil {
		lines = append(lines, fmt.Sprintf("Lease: %s until %s", item.Lease.Owner, item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z")))
	}
	if item.Jira != nil {
		lines = append(lines, fmt.Sprintf("Jira: %s", item.Jira.Key))
	}
	if len(item.DependsOn) > 0 {
		lines = append(lines, fmt.Sprintf("Depends On: %s", strings.Join(item.DependsOn, ", ")))
	}
	return strings.Join(lines, "\n")
}

func ItemShowPrompt(item domain.Item) string {
	lines := []string{
		"ID: " + item.ID,
		"Kind: " + string(item.Kind),
		"Title: " + item.Title,
		"Status: " + string(item.Status),
		fmt.Sprintf("Priority: %d", item.Priority),
		"Goal: " + item.Goal,
		"Summary: " + item.Summary,
		"Next: " + item.NextAction,
	}
	if item.Lease != nil {
		lines = append(lines, "Lease: "+item.Lease.Owner+" until "+item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"))
	}
	if item.Jira != nil {
		lines = append(lines, "Jira: "+item.Jira.Key)
	}
	if len(item.DependsOn) > 0 {
		lines = append(lines, "Depends On: "+strings.Join(item.DependsOn, ", "))
	}
	return strings.Join(lines, "\n")
}

func ItemWithHistoryBrief(item domain.Item, events []domain.Event) string {
	base := ItemShowBrief(item)
	if len(events) == 0 {
		return base + "\nHistory: none"
	}
	lines := []string{base, "History:"}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("  %s %-18s %-8s %s", event.At.Format("2006-01-02T15:04:05Z"), event.Type, event.Actor, event.Summary))
	}
	return strings.Join(lines, "\n")
}

func ItemWithHistoryPrompt(item domain.Item, events []domain.Event) string {
	base := ItemShowPrompt(item)
	if len(events) == 0 {
		return base + "\nHistory: none"
	}
	lines := []string{base, "History:"}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%s %s %s %s", event.At.Format("2006-01-02T15:04:05Z"), event.Type, event.Actor, event.Summary))
	}
	return strings.Join(lines, "\n")
}

func ItemUpdatedBrief(item domain.Item) string {
	return fmt.Sprintf("updated %s [%s] %s", item.ID, item.Status, item.Summary)
}

func ItemUpdatedPrompt(item domain.Item) string {
	lines := []string{
		"Status: updated",
		"ID: " + item.ID,
		"State: " + string(item.Status),
		"Summary: " + item.Summary,
	}
	if item.NextAction != "" {
		lines = append(lines, "Next: "+item.NextAction)
	}
	return strings.Join(lines, "\n")
}

func ItemDoneBrief(item domain.Item) string {
	return fmt.Sprintf("done %s %s", item.ID, item.Title)
}

func ItemDonePrompt(item domain.Item) string {
	return strings.Join([]string{
		"Status: done",
		"ID: " + item.ID,
		"Title: " + item.Title,
		"Summary: " + item.Summary,
	}, "\n")
}

func ItemBlockedBrief(item domain.Item) string {
	return fmt.Sprintf("blocked %s %s", item.ID, item.Summary)
}

func ItemBlockedPrompt(item domain.Item) string {
	lines := []string{
		"Status: blocked",
		"ID: " + item.ID,
		"Summary: " + item.Summary,
	}
	if item.NextAction != "" {
		lines = append(lines, "Next: "+item.NextAction)
	}
	if len(item.DependsOn) > 0 {
		lines = append(lines, "Depends On: "+strings.Join(item.DependsOn, ", "))
	}
	return strings.Join(lines, "\n")
}

func ItemUnblockedBrief(item domain.Item) string {
	return fmt.Sprintf("unblocked %s [%s] %s", item.ID, item.Status, item.Summary)
}

func ItemUnblockedPrompt(item domain.Item) string {
	lines := []string{
		"Status: unblocked",
		"ID: " + item.ID,
		"State: " + string(item.Status),
		"Summary: " + item.Summary,
	}
	if item.NextAction != "" {
		lines = append(lines, "Next: "+item.NextAction)
	}
	return strings.Join(lines, "\n")
}

func ItemTakenBrief(item domain.Item) string {
	return fmt.Sprintf("claimed %s by %s until %s", item.ID, item.Lease.Owner, item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"))
}

func ItemTakenPrompt(item domain.Item) string {
	return strings.Join([]string{
		"Status: claimed",
		"ID: " + item.ID,
		"Owner: " + item.Lease.Owner,
		"Expires: " + item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"),
	}, "\n")
}

func ItemReleasedBrief(item domain.Item) string {
	return fmt.Sprintf("released %s", item.ID)
}

func ItemReleasedPrompt(item domain.Item) string {
	return strings.Join([]string{
		"Status: released",
		"ID: " + item.ID,
		"Summary: " + item.Summary,
	}, "\n")
}

func ItemHandedOffBrief(item domain.Item) string {
	return fmt.Sprintf("handed off %s to %s until %s", item.ID, item.Lease.Owner, item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"))
}

func ItemHandedOffPrompt(item domain.Item) string {
	lines := []string{
		"Status: handoff",
		"ID: " + item.ID,
		"To: " + item.Lease.Owner,
		"Expires: " + item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"),
		"Summary: " + item.Summary,
	}
	if item.NextAction != "" {
		lines = append(lines, "Next: "+item.NextAction)
	}
	return strings.Join(lines, "\n")
}

func ItemReopenedBrief(item domain.Item) string {
	return fmt.Sprintf("reopened %s [%s] %s", item.ID, item.Status, item.Summary)
}

func ItemReopenedPrompt(item domain.Item) string {
	lines := []string{
		"Status: reopened",
		"ID: " + item.ID,
		"State: " + string(item.Status),
		"Summary: " + item.Summary,
		"Next: " + item.NextAction,
	}
	return strings.Join(lines, "\n")
}

func ItemLinkedBrief(item domain.Item, dependencyID string) string {
	return fmt.Sprintf("linked %s depends_on %s", item.ID, dependencyID)
}

func ItemLinkedPrompt(item domain.Item, dependencyID string) string {
	return strings.Join([]string{
		"Status: linked",
		"ID: " + item.ID,
		"Depends On: " + dependencyID,
	}, "\n")
}

func NextItemBrief(result store.NextItemResult) string {
	item := result.Item
	lines := []string{
		fmt.Sprintf("Next: %s", item.ID),
		fmt.Sprintf("Reason: %s", result.Reason),
		fmt.Sprintf("Title: %s", item.Title),
		fmt.Sprintf("Status: %s", item.Status),
		fmt.Sprintf("Priority: %d", item.Priority),
		fmt.Sprintf("Next Action: %s", item.NextAction),
	}
	if len(result.WaitingOn) > 0 {
		lines = append(lines, fmt.Sprintf("Waiting On: %s", strings.Join(result.WaitingOn, ", ")))
	}
	return strings.Join(lines, "\n")
}

func NextItemPrompt(result store.NextItemResult) string {
	item := result.Item
	lines := []string{
		"ID: " + item.ID,
		"Reason: " + result.Reason,
		"Title: " + item.Title,
		"Status: " + string(item.Status),
		fmt.Sprintf("Priority: %d", item.Priority),
		"Next: " + item.NextAction,
	}
	if len(result.WaitingOn) > 0 {
		lines = append(lines, "Waiting On: "+strings.Join(result.WaitingOn, ", "))
	}
	return strings.Join(lines, "\n")
}

func InboxBrief(entries []store.InboxEntry) string {
	if len(entries) == 0 {
		return "Inbox is empty."
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		owner := "-"
		if entry.Item.Lease != nil {
			owner = entry.Item.Lease.Owner
		}
		line := fmt.Sprintf("%-8s %-12s %-12s p%d owner=%-8s %s", entry.Reason, entry.Item.ID, entry.Item.Status, entry.Item.Priority, owner, entry.Item.Title)
		if len(entry.WaitingOn) > 0 {
			line += fmt.Sprintf(" (waiting on %s)", strings.Join(entry.WaitingOn, ", "))
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func InboxPrompt(entries []store.InboxEntry) string {
	if len(entries) == 0 {
		return "Inbox: empty"
	}
	lines := []string{fmt.Sprintf("Inbox: %d", len(entries))}
	for _, entry := range entries {
		line := fmt.Sprintf("%s %s %s %s", entry.Reason, entry.Item.ID, entry.Item.Status, entry.Item.Title)
		if len(entry.WaitingOn) > 0 {
			line += " waiting_on=" + strings.Join(entry.WaitingOn, ",")
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ChangesBrief(events []domain.Event) string {
	if len(events) == 0 {
		return "No changes found."
	}
	lines := make([]string, 0, len(events))
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%s %-12s %-18s %-8s %s", event.At.Format("2006-01-02T15:04:05Z"), event.ItemID, event.Type, event.Actor, event.Summary))
	}
	return strings.Join(lines, "\n")
}

func ChangesPrompt(events []domain.Event) string {
	if len(events) == 0 {
		return "Changes: none"
	}
	lines := []string{fmt.Sprintf("Changes: %d", len(events))}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%s %s %s %s", event.ItemID, event.Type, event.Actor, event.Summary))
	}
	return strings.Join(lines, "\n")
}

func ReadyBrief(entries []store.ReadyEntry) string {
	if len(entries) == 0 {
		return "No ready items found."
	}
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		owner := "-"
		if entry.Item.Lease != nil {
			owner = entry.Item.Lease.Owner
		}
		lines = append(lines, fmt.Sprintf("%-9s %-12s %-12s p%d owner=%-8s %s", entry.Reason, entry.Item.ID, entry.Item.Status, entry.Item.Priority, owner, entry.Item.Title))
	}
	return strings.Join(lines, "\n")
}

func ReadyPrompt(entries []store.ReadyEntry) string {
	if len(entries) == 0 {
		return "Ready: none"
	}
	lines := []string{fmt.Sprintf("Ready: %d", len(entries))}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s %s %s %s", entry.Reason, entry.Item.ID, entry.Item.Status, entry.Item.Title))
	}
	return strings.Join(lines, "\n")
}

func JiraStatusMapBrief(result store.JiraStatusMapResult) string {
	lines := []string{
		fmt.Sprintf("Enabled: %t", result.Enabled),
		fmt.Sprintf("Base URL: %s", fallbackValue(result.BaseURL)),
		fmt.Sprintf("Project: %s", fallbackValue(result.Project)),
	}
	if len(result.Entries) == 0 {
		lines = append(lines, "Mappings: none")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Mappings:")
	for _, entry := range result.Entries {
		lines = append(lines, fmt.Sprintf("  %s -> %s", entry.JiraStatus, entry.LocalStatus))
	}
	return strings.Join(lines, "\n")
}

func JiraStatusMapPrompt(result store.JiraStatusMapResult) string {
	lines := []string{
		fmt.Sprintf("Enabled: %t", result.Enabled),
		"Base URL: " + fallbackValue(result.BaseURL),
		"Project: " + fallbackValue(result.Project),
	}
	if len(result.Entries) == 0 {
		lines = append(lines, "Mappings: none")
		return strings.Join(lines, "\n")
	}
	for _, entry := range result.Entries {
		lines = append(lines, fmt.Sprintf("Map: %s -> %s", entry.JiraStatus, entry.LocalStatus))
	}
	return strings.Join(lines, "\n")
}

func JiraTransitionsBrief(result store.JiraTransitionsResult) string {
	lines := []string{
		fmt.Sprintf("ID: %s", result.Item.ID),
		fmt.Sprintf("Jira: %s", result.JiraKey),
		fmt.Sprintf("Local Status: %s", result.Item.Status),
		fmt.Sprintf("Remote Status: %s", fallbackValue(result.RemoteStatus)),
		fmt.Sprintf("Desired Jira Status: %s", fallbackValue(result.DesiredStatus)),
	}
	if result.CanTransition {
		lines = append(lines, fmt.Sprintf("Matching Transition: %s", fallbackValue(result.MatchingID)))
	} else {
		lines = append(lines, "Matching Transition: none")
	}
	if len(result.Available) == 0 {
		lines = append(lines, "Available Transitions: none")
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Available Transitions:")
	for _, transition := range result.Available {
		line := fmt.Sprintf("  %s %-14s -> %s", transition.ID, fallbackValue(transition.Name), fallbackValue(transition.To))
		if transition.MatchesDesired {
			line += " [matches desired]"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func JiraTransitionsPrompt(result store.JiraTransitionsResult) string {
	lines := []string{
		"ID: " + result.Item.ID,
		"Jira: " + result.JiraKey,
		"Local Status: " + string(result.Item.Status),
		"Remote Status: " + fallbackValue(result.RemoteStatus),
		"Desired Jira Status: " + fallbackValue(result.DesiredStatus),
		fmt.Sprintf("Can Transition: %t", result.CanTransition),
	}
	if result.MatchingID != "" {
		lines = append(lines, "Matching Transition: "+result.MatchingID)
	}
	for _, transition := range result.Available {
		line := fmt.Sprintf("Transition: %s %s -> %s", transition.ID, fallbackValue(transition.Name), fallbackValue(transition.To))
		if transition.MatchesDesired {
			line += " matches_desired=true"
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func JiraSearchBrief(result store.JiraSearchResult) string {
	lines := []string{
		fmt.Sprintf("Project: %s", fallbackValue(result.Project)),
		fmt.Sprintf("Query: %s", fallbackValue(result.Query)),
		fmt.Sprintf("Matches: %d", len(result.Issues)),
	}
	if len(result.Issues) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Issues:")
	for _, issue := range result.Issues {
		lines = append(lines, fmt.Sprintf("  %-10s %-12s %-10s %-8s %s", issue.Key, fallbackValue(issue.Status), fallbackValue(issue.IssueType), fallbackValue(issue.Priority), fallbackValue(issue.Summary)))
	}
	return strings.Join(lines, "\n")
}

func JiraSearchPrompt(result store.JiraSearchResult) string {
	lines := []string{
		"Project: " + fallbackValue(result.Project),
		"Query: " + fallbackValue(result.Query),
		fmt.Sprintf("Matches: %d", len(result.Issues)),
	}
	for _, issue := range result.Issues {
		lines = append(lines, fmt.Sprintf("%s status=%s type=%s priority=%s updated=%s summary=%s", issue.Key, fallbackValue(issue.Status), fallbackValue(issue.IssueType), fallbackValue(issue.Priority), fallbackValue(issue.Updated), fallbackValue(issue.Summary)))
	}
	return strings.Join(lines, "\n")
}

func fallbackValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return value
}
