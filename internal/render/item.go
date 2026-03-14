package render

import (
	"fmt"
	"sort"
	"strings"

	"ajentwork/internal/domain"
	"ajentwork/internal/store"
)

type ItemRelationSummary struct {
	Parent   string   `json:"parent,omitempty"`
	Children []string `json:"children,omitempty"`
	Blocks   []string `json:"blocks,omitempty"`
}

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

func ItemShowBrief(item domain.Item, relations ItemRelationSummary) string {
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
	lines = appendContextLines(lines, item)
	lines = appendCheckpointLines(lines, item)
	if item.Lease != nil {
		lines = append(lines, fmt.Sprintf("Lease: %s until %s", item.Lease.Owner, item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z")))
	}
	if item.Jira != nil {
		lines = append(lines, fmt.Sprintf("Jira: %s", item.Jira.Key))
	}
	if item.ParentID != "" {
		lines = append(lines, fmt.Sprintf("Parent: %s", item.ParentID))
	}
	if len(item.DependsOn) > 0 {
		lines = append(lines, fmt.Sprintf("Depends On: %s", strings.Join(item.DependsOn, ", ")))
	}
	if len(relations.Children) > 0 {
		lines = append(lines, fmt.Sprintf("Children: %s", strings.Join(relations.Children, ", ")))
	}
	if len(relations.Blocks) > 0 {
		lines = append(lines, fmt.Sprintf("Blocks: %s", strings.Join(relations.Blocks, ", ")))
	}
	return strings.Join(lines, "\n")
}

func ItemShowPrompt(item domain.Item, relations ItemRelationSummary) string {
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
	lines = appendContextLines(lines, item)
	lines = appendCheckpointLines(lines, item)
	if item.Lease != nil {
		lines = append(lines, "Lease: "+item.Lease.Owner+" until "+item.Lease.ExpiresAt.Format("2006-01-02T15:04:05Z"))
	}
	if item.Jira != nil {
		lines = append(lines, "Jira: "+item.Jira.Key)
	}
	if item.ParentID != "" {
		lines = append(lines, "Parent: "+item.ParentID)
	}
	if len(item.DependsOn) > 0 {
		lines = append(lines, "Depends On: "+strings.Join(item.DependsOn, ", "))
	}
	if len(relations.Children) > 0 {
		lines = append(lines, "Children: "+strings.Join(relations.Children, ", "))
	}
	if len(relations.Blocks) > 0 {
		lines = append(lines, "Blocks: "+strings.Join(relations.Blocks, ", "))
	}
	return strings.Join(lines, "\n")
}

func ArtifactsSectionBrief(artifacts []domain.Artifact) string {
	if len(artifacts) == 0 {
		return "Artifacts: none"
	}
	lines := []string{"Artifacts:"}
	for _, artifact := range artifacts {
		line := fmt.Sprintf("  %s %-7s %s", artifact.CreatedAt.Format("2006-01-02T15:04:05Z"), artifact.Kind, artifact.Summary)
		if artifact.Kind == domain.ArtifactKindReceipt && artifact.ExitCode != nil {
			line += fmt.Sprintf(" (exit=%d)", *artifact.ExitCode)
		}
		if artifact.StoredPath != "" {
			line += fmt.Sprintf(" -> %s", artifact.StoredPath)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ArtifactsSectionPrompt(artifacts []domain.Artifact) string {
	if len(artifacts) == 0 {
		return "Artifacts: none"
	}
	lines := []string{"Artifacts:"}
	for _, artifact := range artifacts {
		line := fmt.Sprintf("%s %s %s", artifact.ID, artifact.Kind, artifact.Summary)
		if artifact.Kind == domain.ArtifactKindReceipt && artifact.ExitCode != nil {
			line += fmt.Sprintf(" exit=%d", *artifact.ExitCode)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ArtifactsBrief(artifacts []domain.Artifact) string {
	if len(artifacts) == 0 {
		return "No artifacts found."
	}
	lines := make([]string, 0, len(artifacts))
	for _, artifact := range artifacts {
		line := fmt.Sprintf("%s %-12s %-7s %s", artifact.CreatedAt.Format("2006-01-02T15:04:05Z"), artifact.ID, artifact.Kind, artifact.Summary)
		if artifact.Kind == domain.ArtifactKindReceipt && artifact.ExitCode != nil {
			line += fmt.Sprintf(" (exit=%d)", *artifact.ExitCode)
		}
		if artifact.StoredPath != "" {
			line += fmt.Sprintf(" -> %s", artifact.StoredPath)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ArtifactsPrompt(artifacts []domain.Artifact) string {
	if len(artifacts) == 0 {
		return "Artifacts: none"
	}
	lines := []string{fmt.Sprintf("Artifacts: %d", len(artifacts))}
	for _, artifact := range artifacts {
		line := fmt.Sprintf("%s %s %s", artifact.ID, artifact.Kind, artifact.Summary)
		if artifact.ExitCode != nil {
			line += fmt.Sprintf(" exit=%d", *artifact.ExitCode)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func ArtifactAttachedBrief(artifact domain.Artifact) string {
	return fmt.Sprintf("attached %s to %s", artifact.ID, artifact.ItemID)
}

func ArtifactAttachedPrompt(artifact domain.Artifact) string {
	lines := []string{
		"Status: attached",
		"Item: " + artifact.ItemID,
		"Artifact: " + artifact.ID,
		"Kind: " + string(artifact.Kind),
		"Summary: " + artifact.Summary,
	}
	if artifact.StoredPath != "" {
		lines = append(lines, "Stored: "+artifact.StoredPath)
	}
	return strings.Join(lines, "\n")
}

func appendContextLines(lines []string, item domain.Item) []string {
	if len(item.Acceptance) > 0 {
		lines = append(lines, "Acceptance: "+strings.Join(item.Acceptance, "; "))
	}
	if len(item.Constraints) > 0 {
		lines = append(lines, "Constraints: "+strings.Join(item.Constraints, "; "))
	}
	if len(item.Risks) > 0 {
		lines = append(lines, "Risks: "+strings.Join(item.Risks, "; "))
	}
	if len(item.RelevantFiles) > 0 {
		lines = append(lines, "Relevant Files: "+strings.Join(item.RelevantFiles, ", "))
	}
	if len(item.Verification) > 0 {
		lines = append(lines, "Verification: "+strings.Join(item.Verification, "; "))
	}
	return lines
}

func appendCheckpointLines(lines []string, item domain.Item) []string {
	if item.Checkpoint == nil {
		return lines
	}
	lines = append(lines, "Checkpoint: "+item.Checkpoint.Summary)
	if len(item.Checkpoint.Risks) > 0 {
		lines = append(lines, "Checkpoint Risks: "+strings.Join(item.Checkpoint.Risks, "; "))
	}
	if len(item.Checkpoint.Verify) > 0 {
		lines = append(lines, "Checkpoint Verify: "+strings.Join(item.Checkpoint.Verify, "; "))
	}
	lines = append(lines, "Checkpoint At: "+item.Checkpoint.CreatedAt.Format("2006-01-02T15:04:05Z"))
	return lines
}

func ItemWithHistoryBrief(item domain.Item, events []domain.Event) string {
	return ItemWithHistoryBriefAndRelations(item, ItemRelationSummary{}, events)
}

func ItemWithHistoryBriefAndRelations(item domain.Item, relations ItemRelationSummary, events []domain.Event) string {
	base := ItemShowBrief(item, relations)
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
	return ItemWithHistoryPromptAndRelations(item, ItemRelationSummary{}, events)
}

func ItemWithHistoryPromptAndRelations(item domain.Item, relations ItemRelationSummary, events []domain.Event) string {
	base := ItemShowPrompt(item, relations)
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
	if item.Checkpoint != nil {
		lines = append(lines, "Checkpoint: "+item.Checkpoint.Summary)
	}
	return strings.Join(lines, "\n")
}

func ItemCheckpointedBrief(item domain.Item) string {
	return fmt.Sprintf("checkpointed %s %s", item.ID, item.Summary)
}

func ItemCheckpointedPrompt(item domain.Item) string {
	lines := []string{
		"Status: checkpointed",
		"ID: " + item.ID,
		"Summary: " + item.Summary,
	}
	if item.NextAction != "" {
		lines = append(lines, "Next: "+item.NextAction)
	}
	if item.Checkpoint != nil {
		if len(item.Checkpoint.Risks) > 0 {
			lines = append(lines, "Risks: "+strings.Join(item.Checkpoint.Risks, "; "))
		}
		if len(item.Checkpoint.Verify) > 0 {
			lines = append(lines, "Verify: "+strings.Join(item.Checkpoint.Verify, "; "))
		}
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

func ItemParentLinkedBrief(item domain.Item, parentID string) string {
	return fmt.Sprintf("linked %s parent %s", item.ID, parentID)
}

func ItemParentLinkedPrompt(item domain.Item, parentID string) string {
	return strings.Join([]string{
		"Status: linked parent",
		"ID: " + item.ID,
		"Parent: " + parentID,
	}, "\n")
}

func ItemUnlinkedBrief(item domain.Item, dependencyID string) string {
	return fmt.Sprintf("unlinked %s depends_on %s", item.ID, dependencyID)
}

func ItemParentUnlinkedBrief(item domain.Item) string {
	return fmt.Sprintf("unlinked %s parent", item.ID)
}

func ItemUnlinkedPrompt(item domain.Item, relationLabel, targetID string) string {
	lines := []string{
		"Status: unlinked",
		"ID: " + item.ID,
		"Relation: " + relationLabel,
	}
	if targetID != "" {
		lines = append(lines, "Target: "+targetID)
	}
	return strings.Join(lines, "\n")
}

func ItemRelations(item domain.Item, items []domain.Item) ItemRelationSummary {
	relations := ItemRelationSummary{
		Parent: item.ParentID,
	}
	for _, other := range items {
		if other.ParentID == item.ID {
			relations.Children = append(relations.Children, other.ID)
		}
		for _, depID := range other.DependsOn {
			if depID == item.ID {
				relations.Blocks = append(relations.Blocks, other.ID)
				break
			}
		}
	}
	sort.Strings(relations.Children)
	sort.Strings(relations.Blocks)
	return relations
}

func ItemSearchBrief(result store.SearchItemsResult) string {
	lines := []string{
		fmt.Sprintf("Query: %s", fallbackValue(result.Query)),
		fmt.Sprintf("Matches: %d", len(result.Items)),
	}
	if result.Status != "" {
		lines = append(lines, "Status Filter: "+result.Status)
	}
	if result.Kind != "" {
		lines = append(lines, "Kind Filter: "+result.Kind)
	}
	if len(result.Items) == 0 {
		return strings.Join(lines, "\n")
	}
	lines = append(lines, "Items:")
	for _, item := range result.Items {
		lines = append(lines, fmt.Sprintf("  %-12s %-12s p%d %-8s %s", item.ID, item.Status, item.Priority, item.Kind, item.Title))
	}
	return strings.Join(lines, "\n")
}

func ItemSearchPrompt(result store.SearchItemsResult) string {
	lines := []string{
		"Query: " + fallbackValue(result.Query),
		fmt.Sprintf("Matches: %d", len(result.Items)),
	}
	for _, item := range result.Items {
		lines = append(lines, fmt.Sprintf("%s %s p%d %s", item.ID, item.Status, item.Priority, item.Title))
	}
	return strings.Join(lines, "\n")
}

func ReportBrief(result store.ReportResult) string {
	lines := []string{
		fmt.Sprintf("Total Items: %d", result.Total),
		"Status Counts:",
	}
	for _, entry := range result.StatusCounts {
		lines = append(lines, fmt.Sprintf("  %-12s %d", entry.Status, entry.Count))
	}
	if result.Agent != "" {
		lines = append(lines, "Agent: "+result.Agent)
	}
	lines = appendReportSections(lines, result)
	return strings.Join(lines, "\n")
}

func ReportPrompt(result store.ReportResult) string {
	lines := []string{
		fmt.Sprintf("Total: %d", result.Total),
	}
	if result.Agent != "" {
		lines = append(lines, "Agent: "+result.Agent)
	}
	for _, entry := range result.StatusCounts {
		lines = append(lines, fmt.Sprintf("Count %s=%d", entry.Status, entry.Count))
	}
	lines = appendPromptReportSections(lines, result)
	return strings.Join(lines, "\n")
}

func appendReportSections(lines []string, result store.ReportResult) []string {
	lines = append(lines, formatInboxSection("Owned", result.Owned)...)
	lines = append(lines, formatReadySection("Ready", result.Ready)...)
	lines = append(lines, formatInboxSection("Waiting", result.Waiting)...)
	lines = append(lines, formatRecentSection("Recent", result.Recent)...)
	return lines
}

func appendPromptReportSections(lines []string, result store.ReportResult) []string {
	lines = append(lines, formatPromptInboxSection("Owned", result.Owned)...)
	lines = append(lines, formatPromptReadySection("Ready", result.Ready)...)
	lines = append(lines, formatPromptInboxSection("Waiting", result.Waiting)...)
	lines = append(lines, formatPromptRecentSection("Recent", result.Recent)...)
	return lines
}

func formatInboxSection(label string, entries []store.InboxEntry) []string {
	if len(entries) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, entry := range entries {
		line := fmt.Sprintf("  %-12s %-12s %s", entry.Item.ID, entry.Item.Status, entry.Item.Title)
		if len(entry.WaitingOn) > 0 {
			line += fmt.Sprintf(" (waiting on %s)", strings.Join(entry.WaitingOn, ", "))
		}
		lines = append(lines, line)
	}
	return lines
}

func formatReadySection(label string, entries []store.ReadyEntry) []string {
	if len(entries) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("  %-12s %-12s %s", entry.Item.ID, entry.Item.Status, entry.Item.Title))
	}
	return lines
}

func formatRecentSection(label string, events []domain.Event) []string {
	if len(events) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("  %s %-12s %-12s %s", event.ItemID, event.Type, event.Actor, event.Summary))
	}
	return lines
}

func formatPromptInboxSection(label string, entries []store.InboxEntry) []string {
	if len(entries) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s %s %s", entry.Item.ID, entry.Item.Status, entry.Item.Title))
	}
	return lines
}

func formatPromptReadySection(label string, entries []store.ReadyEntry) []string {
	if len(entries) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("%s %s %s", entry.Item.ID, entry.Item.Status, entry.Item.Title))
	}
	return lines
}

func formatPromptRecentSection(label string, events []domain.Event) []string {
	if len(events) == 0 {
		return []string{label + ": none"}
	}
	lines := []string{label + ":"}
	for _, event := range events {
		lines = append(lines, fmt.Sprintf("%s %s %s", event.ItemID, event.Type, event.Summary))
	}
	return lines
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
