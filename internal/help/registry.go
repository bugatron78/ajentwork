package help

import (
	"fmt"
	"sort"
	"strings"
)

type Registry struct {
	root      Doc
	commands  map[string]Doc
	workflows map[string]WorkflowDoc
	examples  map[string]ExampleSet
	glossary  map[string]GlossaryEntry
}

func DefaultRegistry() Registry {
	root := Doc{
		Name:    "aj",
		Summary: "agent work tracker with optional Jira sync",
		Purpose: "Track agent work locally in a compact, git-friendly format with built-in help, workflows, and optional Jira interoperability.",
		Usage:   "aj <command> [options]",
		Related: []string{"help", "commands", "workflows", "examples", "glossary"},
	}

	commands := []Doc{
		{
			Name:    "new",
			Summary: "create a local work item",
			Purpose: "Create a new local aj work item with a compact snapshot and an initial created event.",
			Usage:   "aj new --kind <kind> --title <title> --goal <goal> --next <action> [--priority 2]",
			Options: []OptionDoc{
				{Name: "--kind <kind>", Description: "required item kind: bug, feature, task, spike, or epic"},
				{Name: "--title <title>", Description: "required short work item title"},
				{Name: "--goal <goal>", Description: "required problem statement or desired outcome"},
				{Name: "--next <action>", Description: "required immediate next action for the agent"},
				{Name: "--priority <0-4>", Description: "priority where 0 is highest and 4 is lowest; default 2"},
			},
			Examples: []ExampleDoc{
				{Label: "Create a bug", Command: "aj new --kind bug --title \"Fix cache invalidation\" --goal \"restore correct invalidation\" --next \"inspect update path\""},
			},
			Related:        []string{"ls", "show", "init"},
			WorkflowTags:   []string{"core", "create"},
			SearchKeywords: []string{"create", "item", "ticket"},
		},
		{
			Name:    "ls",
			Summary: "list local work items",
			Purpose: "List compact local work item summaries sorted by priority and recency.",
			Usage:   "aj ls",
			Examples: []ExampleDoc{
				{Label: "List local items", Command: "aj ls"},
			},
			Related:        []string{"show", "new"},
			WorkflowTags:   []string{"core", "list"},
			SearchKeywords: []string{"list", "items", "work"},
		},
		{
			Name:    "show",
			Summary: "show a compact work item summary",
			Purpose: "Show the compact current snapshot for a single local work item, with optional recent event history.",
			Usage:   "aj show <id> [--history] [--limit <n>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--history", Description: "append recent event history after the current snapshot"},
				{Name: "--limit <n>", Description: "optional max number of history events to show when --history is used; default 5"},
			},
			Examples: []ExampleDoc{
				{Label: "Show one item", Command: "aj show W-8F3K2P1Q"},
				{Label: "Show one item with history", Command: "aj show W-8F3K2P1Q --history --limit 5"},
			},
			Related:        []string{"ls", "new", "changes"},
			WorkflowTags:   []string{"core", "inspect", "history"},
			SearchKeywords: []string{"inspect", "details", "summary", "history"},
		},
		{
			Name:    "update",
			Summary: "record progress on a local work item",
			Purpose: "Update a local work item summary and optionally change its next action or active status.",
			Usage:   "aj update <id> --summary <summary> [--next <action>] [--status <status>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required one-line progress summary"},
				{Name: "--next <action>", Description: "optional replacement next action"},
				{Name: "--status <status>", Description: "optional replacement status: todo, in_progress, blocked, or in_review"},
			},
			Examples: []ExampleDoc{
				{Label: "Move into progress", Command: "aj update W-8F3K2P1Q --summary \"started implementation\" --status in_progress --next \"write tests\""},
			},
			Related:        []string{"show", "done", "ls"},
			WorkflowTags:   []string{"core", "progress"},
			SearchKeywords: []string{"progress", "status", "summary"},
		},
		{
			Name:    "block",
			Summary: "mark a local work item blocked",
			Purpose: "Move a local work item into blocked status and optionally attach a dependency that explains what must finish first.",
			Usage:   "aj block <id> --summary <summary> [--on <id>] [--next <action>] [--jira-comment|--no-jira-comment]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required one-line blocker summary"},
				{Name: "--on <id>", Description: "optional dependency item identifier to attach and wait on"},
				{Name: "--next <action>", Description: "optional replacement next action; defaults to waiting on the dependency when --on is used"},
				{Name: "--jira-comment", Description: "post the same summary to the linked Jira issue after blocking"},
				{Name: "--no-jira-comment", Description: "suppress Jira milestone comment even if the repo policy enables it"},
			},
			Examples: []ExampleDoc{
				{Label: "Block on another item", Command: "aj block W-8F3K2P1Q --on W-2M9A1C7L --summary \"waiting on schema decision\""},
			},
			Related:        []string{"unblock", "link", "show"},
			WorkflowTags:   []string{"coordination", "blocked"},
			SearchKeywords: []string{"blocked", "wait", "dependency"},
		},
		{
			Name:    "unblock",
			Summary: "clear blocked status from a local work item",
			Purpose: "Move a blocked item back into active work, usually after a dependency is resolved or external blocker is removed.",
			Usage:   "aj unblock <id> --summary <summary> [--next <action>] [--status <status>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required one-line unblock summary"},
				{Name: "--next <action>", Description: "optional replacement next action"},
				{Name: "--status <status>", Description: "optional replacement status after unblocking; default todo"},
			},
			Examples: []ExampleDoc{
				{Label: "Resume work", Command: "aj unblock W-8F3K2P1Q --summary \"schema approved\" --status in_progress --next \"finish cache updates\""},
			},
			Related:        []string{"block", "update", "show"},
			WorkflowTags:   []string{"coordination", "blocked"},
			SearchKeywords: []string{"unblock", "resume", "waiting"},
		},
		{
			Name:    "done",
			Summary: "complete a local work item",
			Purpose: "Mark a local work item done, record its completion summary, and clear its next action.",
			Usage:   "aj done <id> --summary <summary> [--jira-comment|--no-jira-comment]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required one-line completion summary"},
				{Name: "--jira-comment", Description: "post the same summary to the linked Jira issue after completion"},
				{Name: "--no-jira-comment", Description: "suppress Jira milestone comment even if the repo policy enables it"},
			},
			Examples: []ExampleDoc{
				{Label: "Complete an item", Command: "aj done W-8F3K2P1Q --summary \"tests added and command shipped\""},
			},
			Related:        []string{"update", "show", "ls"},
			WorkflowTags:   []string{"core", "complete"},
			SearchKeywords: []string{"complete", "finish", "close"},
		},
		{
			Name:    "take",
			Summary: "claim a local work item for an agent",
			Purpose: "Attach a temporary lease to a local work item so agents can coordinate ownership and avoid duplicate effort.",
			Usage:   "aj take <id> --agent <name> [--ttl 4h] [--force]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--agent <name>", Description: "required agent name to assign to the lease"},
				{Name: "--ttl <duration>", Description: "optional lease duration such as 30m or 4h; default 4h"},
				{Name: "--force", Description: "override a currently active lease"},
			},
			Examples: []ExampleDoc{
				{Label: "Claim an item", Command: "aj take W-8F3K2P1Q --agent coder-1 --ttl 2h"},
			},
			Related:        []string{"release", "show", "ls"},
			Safety:         []string{"Use --force only when the current lease is stale or an explicit handoff has happened."},
			WorkflowTags:   []string{"coordination", "claim"},
			SearchKeywords: []string{"lease", "claim", "owner"},
		},
		{
			Name:    "release",
			Summary: "clear the active lease from a local work item",
			Purpose: "Remove the current lease from a local work item so another agent can claim it.",
			Usage:   "aj release <id>",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Examples: []ExampleDoc{
				{Label: "Release an item", Command: "aj release W-8F3K2P1Q"},
			},
			Related:        []string{"take", "show", "ls"},
			WorkflowTags:   []string{"coordination", "claim"},
			SearchKeywords: []string{"release", "unclaim", "lease"},
		},
		{
			Name:    "handoff",
			Summary: "transfer a local work item lease to another agent",
			Purpose: "Assign a new lease owner with a handoff summary so another agent can continue the work without ambiguity.",
			Usage:   "aj handoff <id> --to <agent> --summary <summary> [--next <action>] [--ttl 4h] [--jira-comment|--no-jira-comment]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--to <agent>", Description: "required destination agent name"},
				{Name: "--summary <summary>", Description: "required compact handoff summary"},
				{Name: "--next <action>", Description: "optional replacement next action for the receiving agent"},
				{Name: "--ttl <duration>", Description: "optional lease duration such as 30m or 4h; default 4h"},
				{Name: "--jira-comment", Description: "post the same summary to the linked Jira issue after handoff"},
				{Name: "--no-jira-comment", Description: "suppress Jira milestone comment even if the repo policy enables it"},
			},
			Examples: []ExampleDoc{
				{Label: "Hand off for review", Command: "aj handoff W-8F3K2P1Q --to reviewer-1 --summary \"implementation ready for review\" --next \"verify CLI behavior\""},
			},
			Related:        []string{"take", "release", "show"},
			Safety:         []string{"Use handoff when ownership is explicitly changing, rather than force-claiming another agent's active lease."},
			WorkflowTags:   []string{"coordination", "handoff"},
			SearchKeywords: []string{"handoff", "transfer", "lease"},
		},
		{
			Name:    "reopen",
			Summary: "reopen completed work with a fresh next action",
			Purpose: "Move a done item back into active work when follow-up or regression handling is needed.",
			Usage:   "aj reopen <id> --summary <summary> --next <action> [--status <status>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required one-line reopen summary"},
				{Name: "--next <action>", Description: "required next action after reopening"},
				{Name: "--status <status>", Description: "optional replacement status after reopening; default todo"},
			},
			Examples: []ExampleDoc{
				{Label: "Reopen a regression", Command: "aj reopen W-8F3K2P1Q --summary \"regression reproduced\" --next \"add a failing test\" --status in_progress"},
			},
			Related:        []string{"done", "update", "show"},
			WorkflowTags:   []string{"core", "reopen"},
			SearchKeywords: []string{"reopen", "regression", "done"},
		},
		{
			Name:    "next",
			Summary: "recommend the next work item to focus on",
			Purpose: "Recommend the most relevant next item, preferring work already leased to the given agent and otherwise falling back to the highest-priority available item.",
			Usage:   "aj next [--agent <name>]",
			Options: []OptionDoc{
				{Name: "--agent <name>", Description: "optional agent name; if provided, owned work is recommended first"},
			},
			Examples: []ExampleDoc{
				{Label: "Recommend work for an agent", Command: "aj next --agent coder-1"},
			},
			Related:        []string{"inbox", "take", "ls"},
			WorkflowTags:   []string{"coordination", "recommend"},
			SearchKeywords: []string{"recommend", "queue", "prioritize"},
		},
		{
			Name:    "inbox",
			Summary: "show work that needs attention",
			Purpose: "Show a compact queue of owned, stale, waiting, and available work items so an agent can decide what to act on next.",
			Usage:   "aj inbox [--agent <name>]",
			Options: []OptionDoc{
				{Name: "--agent <name>", Description: "optional agent name; if provided, owned items for that agent are highlighted"},
			},
			Examples: []ExampleDoc{
				{Label: "Show an agent inbox", Command: "aj inbox --agent coder-1"},
			},
			Related:        []string{"next", "take", "release", "ls"},
			WorkflowTags:   []string{"coordination", "queue"},
			SearchKeywords: []string{"attention", "queue", "owned"},
		},
		{
			Name:    "link",
			Summary: "link one work item as depending on another",
			Purpose: "Record a dependency edge so aj can understand that an item is waiting for another item to be completed before it is ready.",
			Usage:   "aj link <id> --depends-on <id>",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier to update", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--depends-on <id>", Description: "required dependency item identifier"},
			},
			Examples: []ExampleDoc{
				{Label: "Link a dependency", Command: "aj link W-CHILD --depends-on W-PARENT"},
			},
			Related:        []string{"show", "next", "inbox"},
			WorkflowTags:   []string{"coordination", "dependency"},
			SearchKeywords: []string{"depends_on", "dependency", "waiting"},
		},
		{
			Name:    "changes",
			Summary: "show recent events across the repo or for one item",
			Purpose: "List compact recent changes from the append-only event log so agents can understand what happened without inspecting each item manually.",
			Usage:   "aj changes [--item <id>] [--since <rfc3339>] [--limit <n>]",
			Options: []OptionDoc{
				{Name: "--item <id>", Description: "optional work item identifier to limit the feed to one item"},
				{Name: "--since <rfc3339>", Description: "optional lower time bound for returned events"},
				{Name: "--limit <n>", Description: "optional max number of events to return; default 20"},
			},
			Examples: []ExampleDoc{
				{Label: "Show recent repo changes", Command: "aj changes"},
				{Label: "Show one item's changes", Command: "aj changes --item W-8F3K2P1Q"},
			},
			Related:        []string{"show", "inbox", "next"},
			WorkflowTags:   []string{"coordination", "history"},
			SearchKeywords: []string{"history", "events", "activity"},
		},
		{
			Name:    "ready",
			Summary: "list work items that can start now",
			Purpose: "Show actionable items whose dependencies are satisfied and whose lease state does not block the current agent.",
			Usage:   "aj ready [--agent <name>]",
			Options: []OptionDoc{
				{Name: "--agent <name>", Description: "optional agent name; owned items for that agent are included and prioritized"},
			},
			Examples: []ExampleDoc{
				{Label: "Show all ready work", Command: "aj ready"},
				{Label: "Show ready work for an agent", Command: "aj ready --agent coder-1"},
			},
			Related:        []string{"next", "inbox", "take"},
			WorkflowTags:   []string{"coordination", "ready"},
			SearchKeywords: []string{"actionable", "ready", "unblocked"},
		},
		{
			Name:    "jira",
			Summary: "import or export Jira issues",
			Purpose: "Use the Jira adapter to pull Jira issues into local aj items, push local items to Jira, link or unlink local items, inspect status mappings and transitions, sync linked items with conflict detection, and send compact milestone comments back to Jira.",
			Usage:   "aj jira <pull|push|link|unlink|status-map|transitions|sync|comment> ...",
			Options: []OptionDoc{
				{Name: "pull <key>", Description: "import a Jira issue into aj unless it is already linked"},
				{Name: "push <id>", Description: "create a Jira issue from a local aj item unless it is already linked"},
				{Name: "link <id> <key> [--replace]", Description: "attach an existing local item to an existing Jira issue; require --replace to swap an existing link"},
				{Name: "unlink <id> [--force]", Description: "remove the local Jira link; require --force if the item has unsynced Jira state"},
				{Name: "status-map", Description: "show the configured Jira-to-aj status mapping from .aj/config.toml"},
				{Name: "transitions <id>", Description: "inspect the live remote status and available Jira transitions for a linked item"},
				{Name: "sync <id>", Description: "sync a linked local item with Jira; use --dry-run to preview direction and Jira status transitions"},
				{Name: "comment <id> --summary <summary>", Description: "post a compact milestone comment to the linked Jira issue"},
				{Name: "env: AJ_JIRA_EMAIL", Description: "email used for Jira Cloud basic auth"},
				{Name: "env: AJ_JIRA_API_TOKEN", Description: "API token used for Jira Cloud basic auth"},
			},
			Examples: []ExampleDoc{
				{Label: "Import a Jira issue", Command: "aj jira pull ABC-123"},
				{Label: "Push a local item to Jira", Command: "aj jira push W-8F3K2P1Q --project ABC --type Task"},
				{Label: "Link local work to Jira", Command: "aj jira link W-8F3K2P1Q ABC-123"},
				{Label: "Unlink local work from Jira", Command: "aj jira unlink W-8F3K2P1Q"},
				{Label: "Inspect configured status mapping", Command: "aj jira status-map"},
				{Label: "Inspect live transitions", Command: "aj jira transitions W-8F3K2P1Q"},
				{Label: "Preview a sync", Command: "aj jira sync W-8F3K2P1Q --dry-run"},
				{Label: "Post a milestone comment", Command: "aj jira comment W-8F3K2P1Q --summary \"ready for review\""},
				{Label: "Import and claim Jira work", Command: "aj take jira ABC-123 --agent coder-1"},
			},
			Related:        []string{"take", "show", "workflows"},
			Safety:         []string{"Set Jira credentials through environment variables instead of committing secrets into .aj/config.toml.", "Use `aj jira status-map` and `aj jira transitions <id>` before the first sync or whenever local and remote workflow behavior is unclear.", "Use `aj jira unlink --force` only when you intentionally want to drop a dirty or conflicted Jira link.", "Sync may attempt a Jira status transition when local status differs and a mapped transition exists."},
			WorkflowTags:   []string{"jira", "integration"},
			SearchKeywords: []string{"jira", "import", "export", "sync", "comment", "transition", "status map", "workflow", "unlink", "relink"},
		},
		{
			Name:    "init",
			Summary: "create the .aj workspace structure in the current repository",
			Purpose: "Bootstrap a repository for aj by creating the .aj directory tree, default config, cache directory, and artifact directories.",
			Usage:   "aj init [--repo <path>] [--force]",
			Options: []OptionDoc{
				{Name: "--repo <path>", Description: "initialize aj in a specific repository path instead of the current working directory"},
				{Name: "--force", Description: "overwrite an existing config file if present"},
				{Name: "--format brief|json|prompt", Description: "render the result in the requested output format"},
			},
			Examples: []ExampleDoc{
				{Label: "Initialize current repo", Command: "aj init"},
				{Label: "Initialize another repo", Command: "aj --repo /path/to/repo init"},
			},
			Related:        []string{"help", "commands"},
			WorkflowTags:   []string{"core"},
			SearchKeywords: []string{"bootstrap", "setup", "repository"},
		},
		{
			Name:    "help",
			Summary: "show command, workflow, example, or glossary help",
			Purpose: "Render top-level help, per-command help, or topic help for workflows, examples, and glossary entries.",
			Usage:   "aj help [topic] [--format brief|json|prompt]",
			Arguments: []ArgDoc{
				{Name: "topic", Description: "optional command or discovery topic such as init, workflows jira, or glossary lease", Required: false},
			},
			Examples: []ExampleDoc{
				{Label: "Show root help", Command: "aj help"},
				{Label: "Show command help", Command: "aj help init"},
				{Label: "Search help", Command: "aj help search jira"},
			},
			Related:        []string{"commands", "workflows", "examples", "glossary"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"discover", "usage", "docs"},
		},
		{
			Name:    "commands",
			Summary: "list available commands with one-line descriptions",
			Purpose: "Show the main aj command surface in a compact format so agents can quickly discover the next command to inspect.",
			Usage:   "aj commands [--format brief|json|prompt]",
			Examples: []ExampleDoc{
				{Label: "List all commands", Command: "aj commands"},
			},
			Related:        []string{"help", "workflows"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"list", "discover", "surface"},
		},
		{
			Name:    "workflows",
			Summary: "show common multi-step workflows",
			Purpose: "Describe common aj sequences such as initializing a repo, claiming Jira work, or handling blockers.",
			Usage:   "aj workflows [topic] [--format brief|json|prompt]",
			Arguments: []ArgDoc{
				{Name: "topic", Description: "optional workflow topic such as core, claim, blocked, or jira", Required: false},
			},
			Examples: []ExampleDoc{
				{Label: "List workflows", Command: "aj workflows"},
				{Label: "Show Jira workflow", Command: "aj workflows jira"},
			},
			Related:        []string{"examples", "help"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"flow", "process", "steps"},
		},
		{
			Name:    "examples",
			Summary: "show copy-pasteable command examples",
			Purpose: "Provide short example commands for the most common aj tasks and workflows.",
			Usage:   "aj examples [topic] [--format brief|json|prompt]",
			Arguments: []ArgDoc{
				{Name: "topic", Description: "optional example topic such as init, jira, blocked, or handoff", Required: false},
			},
			Examples: []ExampleDoc{
				{Label: "List example topics", Command: "aj examples"},
				{Label: "Show init examples", Command: "aj examples init"},
			},
			Related:        []string{"workflows", "help"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"samples", "snippets", "copy"},
		},
		{
			Name:    "glossary",
			Summary: "define the terms used by aj",
			Purpose: "Explain stable terminology such as lease, sync_state, conflict, and artifact so agents do not have to infer meaning from context.",
			Usage:   "aj glossary [term] [--format brief|json|prompt]",
			Arguments: []ArgDoc{
				{Name: "term", Description: "optional glossary term to inspect", Required: false},
			},
			Examples: []ExampleDoc{
				{Label: "List glossary terms", Command: "aj glossary"},
				{Label: "Define lease", Command: "aj glossary lease"},
			},
			Related:        []string{"help", "workflows"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"terms", "definitions", "vocabulary"},
		},
	}

	workflows := map[string]WorkflowDoc{
		"core": {
			Name:  "core",
			Topic: "create and bootstrap a local aj repository",
			Steps: []string{
				"1. Run `aj init` in the repository root.",
				"2. Run `aj new --kind task --title \"...\" --goal \"...\" --next \"...\"` to create the first item.",
				"3. Use `aj ls` to inspect the local queue.",
				"4. Use `aj show <id>` before updating or handing work off.",
			},
		},
		"jira": {
			Name:  "jira",
			Topic: "import and claim human-created Jira work",
			Steps: []string{
				"1. Set `AJ_JIRA_EMAIL` and `AJ_JIRA_API_TOKEN`, and enable Jira in `.aj/config.toml`.",
				"2. Run `aj take jira ABC-123 --agent coder-1` to import and claim human-created work.",
				"3. Use `aj show <id>` to inspect the normalized local item.",
				"4. Use `aj jira status-map` and `aj jira transitions <id>` to inspect the workflow before syncing.",
				"5. Use `aj update <id> --summary ... --next ...` to record progress, `aj jira comment <id> --summary \"ready for review\"` for milestone updates, or `aj jira sync <id> --dry-run` to preview sync direction and status transitions.",
				"6. Use `aj jira unlink <id>` before relinking to a different Jira issue, or `aj jira link <id> <key> --replace` when that swap is intentional.",
			},
		},
		"blocked": {
			Name:  "blocked",
			Topic: "record and communicate blocked work",
			Steps: []string{
				"1. Use `aj block <id> --on <dependency> --summary \"...\"` when work cannot proceed.",
				"2. Add `--next \"...\"` when the recovery step should be explicit instead of just waiting.",
				"3. Use `aj unblock <id> --summary \"...\" --status in_progress --next \"...\"` when the blocker is cleared.",
			},
		},
		"progress": {
			Name:  "progress",
			Topic: "record progress while keeping next steps explicit",
			Steps: []string{
				"1. Use `aj update <id> --summary \"...\" --status in_progress --next \"...\"` as work starts.",
				"2. Use `aj show <id>` to confirm the compact snapshot still reflects reality.",
				"3. Use `aj done <id> --summary \"...\"` when the item is complete.",
			},
		},
		"claim": {
			Name:  "claim",
			Topic: "claim and release work so agents do not overlap",
			Steps: []string{
				"1. Use `aj take <id> --agent coder-1` before starting work.",
				"2. Use `aj show <id>` to confirm the lease and expiry.",
				"3. Use `aj release <id>` if you are handing the work back without finishing it.",
			},
		},
		"handoff": {
			Name:  "handoff",
			Topic: "transfer ownership without losing context",
			Steps: []string{
				"1. Use `aj show <id>` before handing work off so the current snapshot is accurate.",
				"2. Use `aj handoff <id> --to reviewer-1 --summary \"...\" --next \"...\"` to transfer the lease.",
				"3. Use `aj inbox --agent reviewer-1` or `aj next --agent reviewer-1` so the receiving agent can pick it up immediately.",
			},
		},
		"queue": {
			Name:  "queue",
			Topic: "find the next best item without scanning the whole backlog",
			Steps: []string{
				"1. Use `aj inbox --agent coder-1` to see owned, stale, waiting, and available work.",
				"2. Use `aj next --agent coder-1` to get one recommended item and reason.",
				"3. Use `aj take <id> --agent coder-1` if the recommended item is available and you want to claim it.",
			},
		},
		"dependencies": {
			Name:  "dependencies",
			Topic: "declare dependency edges so waiting work is not recommended too early",
			Steps: []string{
				"1. Use `aj link <id> --depends-on <dependency>` when work cannot start until another item is done.",
				"2. Use `aj show <id>` to confirm the dependency is recorded.",
				"3. Use `aj inbox --agent coder-1` or `aj next --agent coder-1` and let aj mark dependent work as waiting.",
			},
		},
		"history": {
			Name:  "history",
			Topic: "inspect recent activity without scanning every item",
			Steps: []string{
				"1. Use `aj changes` to see the latest repo-wide events.",
				"2. Use `aj changes --item <id>` when you only care about one item's activity.",
				"3. Use `aj show <id>` after reading changes if you need the current snapshot too.",
			},
		},
		"ready": {
			Name:  "ready",
			Topic: "see only work that can actually start right now",
			Steps: []string{
				"1. Use `aj ready` to list actionable items whose dependencies are already satisfied.",
				"2. Use `aj ready --agent coder-1` to include and prioritize work you already own.",
				"3. Use `aj take <id> --agent coder-1` to claim a ready item if it is still available.",
			},
		},
	}

	examples := map[string]ExampleSet{
		"init": {
			Topic: "init",
			Examples: []ExampleDoc{
				{Label: "Initialize current repository", Command: "aj init"},
				{Label: "Initialize a repo from another working directory", Command: "aj --repo /workspace/project init"},
			},
		},
		"new": {
			Topic: "new",
			Examples: []ExampleDoc{
				{Label: "Create a task", Command: "aj new --kind task --title \"Bootstrap CLI\" --goal \"ship the first command set\" --next \"implement item storage\""},
				{Label: "Create a bug", Command: "aj new --kind bug --title \"Fix cache invalidation\" --goal \"restore update path invalidation\" --next \"inspect update handler\" --priority 1"},
			},
		},
		"progress": {
			Topic: "progress",
			Examples: []ExampleDoc{
				{Label: "Update a task", Command: "aj update W-8F3K2P1Q --summary \"started implementation\" --status in_progress --next \"write tests\""},
				{Label: "Complete a task", Command: "aj done W-8F3K2P1Q --summary \"tests added and command shipped\""},
				{Label: "Reopen completed work", Command: "aj reopen W-8F3K2P1Q --summary \"regression reproduced\" --next \"add a failing test\""},
			},
		},
		"blocked": {
			Topic: "blocked",
			Examples: []ExampleDoc{
				{Label: "Block on a dependency", Command: "aj block W-8F3K2P1Q --on W-2M9A1C7L --summary \"waiting on schema decision\""},
				{Label: "Unblock and resume", Command: "aj unblock W-8F3K2P1Q --summary \"schema approved\" --status in_progress --next \"finish cache updates\""},
			},
		},
		"claim": {
			Topic: "claim",
			Examples: []ExampleDoc{
				{Label: "Claim work", Command: "aj take W-8F3K2P1Q --agent coder-1 --ttl 4h"},
				{Label: "Release work", Command: "aj release W-8F3K2P1Q"},
				{Label: "Hand off work", Command: "aj handoff W-8F3K2P1Q --to reviewer-1 --summary \"implementation ready\" --next \"review CLI output\""},
			},
		},
		"queue": {
			Topic: "queue",
			Examples: []ExampleDoc{
				{Label: "Find one next item", Command: "aj next --agent coder-1"},
				{Label: "Show the current inbox", Command: "aj inbox --agent coder-1"},
			},
		},
		"dependencies": {
			Topic: "dependencies",
			Examples: []ExampleDoc{
				{Label: "Record a dependency", Command: "aj link W-CHILD --depends-on W-PARENT"},
				{Label: "Inspect waiting work", Command: "aj inbox --agent coder-1"},
			},
		},
		"history": {
			Topic: "history",
			Examples: []ExampleDoc{
				{Label: "Show repo changes", Command: "aj changes --limit 10"},
				{Label: "Show item changes", Command: "aj changes --item W-8F3K2P1Q"},
			},
		},
		"ready": {
			Topic: "ready",
			Examples: []ExampleDoc{
				{Label: "Show ready work", Command: "aj ready"},
				{Label: "Show ready work for an agent", Command: "aj ready --agent coder-1"},
			},
		},
		"jira": {
			Topic: "jira",
			Examples: []ExampleDoc{
				{Label: "Import and claim Jira work", Command: "aj take jira ABC-123 --agent coder-1"},
				{Label: "Import a Jira issue", Command: "aj jira pull ABC-123"},
				{Label: "Push a local item to Jira", Command: "aj jira push W-8F3K2P1Q --project ABC --type Task"},
				{Label: "Link a local item to Jira", Command: "aj jira link W-8F3K2P1Q ABC-123"},
				{Label: "Unlink a local Jira item", Command: "aj jira unlink W-8F3K2P1Q"},
				{Label: "Show configured Jira mapping", Command: "aj jira status-map"},
				{Label: "Show live Jira transitions", Command: "aj jira transitions W-8F3K2P1Q"},
				{Label: "Preview linked sync", Command: "aj jira sync W-8F3K2P1Q --dry-run"},
				{Label: "Post a Jira milestone comment", Command: "aj jira comment W-8F3K2P1Q --summary \"ready for review\""},
			},
		},
	}

	glossary := map[string]GlossaryEntry{
		"lease": {
			Term:       "lease",
			Definition: "A temporary claim on a work item that shows which agent is actively working it and when that claim expires.",
		},
		"artifact": {
			Term:       "artifact",
			Definition: "A file reference for logs, long notes, or generated output that is too large to keep in the compact item snapshot.",
		},
		"sync_state": {
			Term:       "sync_state",
			Definition: "The current local-versus-remote Jira sync status: clean, dirty_local, dirty_remote, or conflict.",
		},
		"conflict": {
			Term:       "conflict",
			Definition: "A sync state where both the local item and the remote Jira issue changed after the last successful sync and explicit resolution is required.",
		},
		"work item": {
			Term:       "work item",
			Definition: "The compact local unit of work tracked by aj, including its current snapshot and append-only event history.",
		},
	}

	commandMap := make(map[string]Doc, len(commands))
	for _, doc := range commands {
		commandMap[doc.Name] = doc
	}

	return Registry{
		root:      root,
		commands:  commandMap,
		workflows: workflows,
		examples:  examples,
		glossary:  glossary,
	}
}

func (r Registry) Root() Doc {
	return r.root
}

func (r Registry) Command(name string) (Doc, bool) {
	doc, ok := r.commands[name]
	return doc, ok
}

func (r Registry) Commands() []CommandSummary {
	result := make([]CommandSummary, 0, len(r.commands))
	for _, doc := range r.commands {
		group := "Core"
		switch doc.Name {
		case "help", "commands", "workflows", "examples", "glossary":
			group = "Discovery"
		case "jira":
			group = "Jira"
		}

		result = append(result, CommandSummary{
			Name:    doc.Name,
			Summary: doc.Summary,
			Group:   group,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Group == result[j].Group {
			return result[i].Name < result[j].Name
		}
		return result[i].Group < result[j].Group
	})

	return result
}

func (r Registry) Workflow(topic string) (WorkflowDoc, bool) {
	doc, ok := r.workflows[topic]
	return doc, ok
}

func (r Registry) Workflows() []WorkflowDoc {
	keys := make([]string, 0, len(r.workflows))
	for key := range r.workflows {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]WorkflowDoc, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.workflows[key])
	}
	return result
}

func (r Registry) ExampleSet(topic string) (ExampleSet, bool) {
	doc, ok := r.examples[topic]
	return doc, ok
}

func (r Registry) ExampleSets() []ExampleSet {
	keys := make([]string, 0, len(r.examples))
	for key := range r.examples {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]ExampleSet, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.examples[key])
	}
	return result
}

func (r Registry) GlossaryEntry(term string) (GlossaryEntry, bool) {
	entry, ok := r.glossary[term]
	return entry, ok
}

func (r Registry) Glossary() []GlossaryEntry {
	keys := make([]string, 0, len(r.glossary))
	for key := range r.glossary {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]GlossaryEntry, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.glossary[key])
	}
	return result
}

func (r Registry) Search(term string) []SearchResult {
	query := strings.TrimSpace(strings.ToLower(term))
	if query == "" {
		return nil
	}

	var results []SearchResult
	for _, cmd := range r.commands {
		if matches(query, cmd.Name, cmd.Summary, cmd.Purpose, strings.Join(cmd.Related, " "), strings.Join(cmd.WorkflowTags, " "), strings.Join(cmd.SearchKeywords, " ")) {
			results = append(results, SearchResult{Kind: "command", Name: cmd.Name, Summary: cmd.Summary})
		}
	}
	for _, workflow := range r.workflows {
		if matches(query, workflow.Name, workflow.Topic, strings.Join(workflow.Steps, " ")) {
			results = append(results, SearchResult{Kind: "workflow", Name: workflow.Name, Summary: workflow.Topic})
		}
	}
	for _, exampleSet := range r.examples {
		var fragments []string
		for _, example := range exampleSet.Examples {
			fragments = append(fragments, example.Label, example.Command)
		}
		if matches(query, exampleSet.Topic, strings.Join(fragments, " ")) {
			results = append(results, SearchResult{Kind: "example", Name: exampleSet.Topic, Summary: fmt.Sprintf("examples for %s", exampleSet.Topic)})
		}
	}
	for _, entry := range r.glossary {
		if matches(query, entry.Term, entry.Definition) {
			results = append(results, SearchResult{Kind: "glossary", Name: entry.Term, Summary: entry.Definition})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		if results[i].Kind == results[j].Kind {
			return results[i].Name < results[j].Name
		}
		return results[i].Kind < results[j].Kind
	})

	return results
}

func matches(query string, values ...string) bool {
	for _, value := range values {
		if strings.Contains(strings.ToLower(value), query) {
			return true
		}
	}
	return false
}
