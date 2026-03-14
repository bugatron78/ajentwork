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
		Purpose: "Track agent work locally in a compact, git-friendly format with built-in help, workflows, and optional Jira interoperability. Write work items and updates so another agent can pick up the task without rediscovering all the context.",
		Usage:   "aj <command> [options]",
		Related: []string{"help", "commands", "workflows", "examples", "glossary"},
	}

	commands := []Doc{
		{
			Name:    "version",
			Summary: "show the installed aj build version",
			Purpose: "Print the current aj build version so agents and humans can confirm which binary is installed.",
			Usage:   "aj version",
			Examples: []ExampleDoc{
				{Label: "Show version", Command: "aj version"},
				{Label: "Show version with the global flag", Command: "aj --version"},
			},
			Related:        []string{"help", "commands"},
			WorkflowTags:   []string{"discovery"},
			SearchKeywords: []string{"version", "build", "release"},
		},
		{
			Name:    "new",
			Summary: "create a local work item",
			Purpose: "Create a new local aj work item with a compact snapshot and an initial created event. The title, goal, and next action should give another agent enough context to start confidently, and structured fields should capture acceptance criteria, constraints, risks, relevant files, and verification expectations whenever they matter.",
			Usage:   "aj new --kind <kind> --title <title> --goal <goal> --next <action> [--accept <text> ...] [--constraint <text> ...] [--risk <text> ...] [--file <path> ...] [--verify <text> ...] [--priority 2]",
			Options: []OptionDoc{
				{Name: "--kind <kind>", Description: "required item kind: bug, feature, task, spike, or epic"},
				{Name: "--title <title>", Description: "required short work item title"},
				{Name: "--goal <goal>", Description: "required problem statement or desired outcome"},
				{Name: "--next <action>", Description: "required immediate next action for the agent"},
				{Name: "--accept <text>", Description: "optional acceptance criterion; repeat to capture multiple outcomes"},
				{Name: "--constraint <text>", Description: "optional implementation or product constraint; repeat as needed"},
				{Name: "--risk <text>", Description: "optional known risk or uncertainty; repeat as needed"},
				{Name: "--file <path>", Description: "optional relevant file path or area to inspect first; repeat as needed"},
				{Name: "--verify <text>", Description: "optional verification step or evidence expectation; repeat as needed"},
				{Name: "--priority <0-4>", Description: "priority where 0 is highest and 4 is lowest; default 2"},
			},
			Examples: []ExampleDoc{
				{Label: "Create a bug with structured context", Command: "aj new --kind bug --title \"Fix cache invalidation after deletes\" --goal \"restore correct invalidation for delete paths; reproduce with the regression in cache/service_test.go and preserve update-path behavior\" --next \"trace the delete invalidation branch and capture the failing test\" --accept \"delete invalidates stale cache entries\" --accept \"update-path behavior stays correct\" --constraint \"keep cache key format backward compatible during rollout\" --risk \"soft-delete path may share logic with tenant invalidation\" --file cache/service.go --file cache/service_test.go --verify \"run go test ./... and confirm the delete regression passes\""},
			},
			Related:        []string{"ls", "show", "init"},
			Safety:         []string{"Write goal and next so another agent can understand the problem, the important constraints, and the immediate starting point without reopening the whole repo first.", "Use structured fields when they materially change execution quality: acceptance for success criteria, constraints for guardrails, risks for uncertainties, relevant files for starting points, and verification for proof."},
			WorkflowTags:   []string{"core", "create"},
			SearchKeywords: []string{"create", "item", "ticket", "context", "authoring"},
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
			Purpose: "Show the compact current snapshot for a single local work item, including local graph context such as parent, children, dependencies, and blocked downstream work, with optional recent event history.",
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
			Name:    "search",
			Summary: "search local work items by text and optional filters",
			Purpose: "Search local aj work items across titles, summaries, goals, next actions, structured context, relations, and Jira linkage so an agent can quickly find relevant existing work before creating or updating anything.",
			Usage:   "aj search [terms...] [--status <status>] [--kind <kind>] [--limit <n>]",
			Options: []OptionDoc{
				{Name: "--status <status>", Description: "optional status filter: todo, in_progress, blocked, in_review, done, or canceled"},
				{Name: "--kind <kind>", Description: "optional kind filter: bug, feature, task, spike, or epic"},
				{Name: "--limit <n>", Description: "optional max number of matches to return; default 20"},
			},
			Examples: []ExampleDoc{
				{Label: "Search by text", Command: "aj search cache invalidation"},
				{Label: "Find blocked bugs", Command: "aj search regression --status blocked --kind bug"},
			},
			Related:        []string{"ls", "show", "report"},
			Safety:         []string{"Use local search before creating new work so agents can avoid duplicate tickets and recover context from earlier attempts."},
			WorkflowTags:   []string{"discovery", "query"},
			SearchKeywords: []string{"search", "find", "query", "lookup"},
		},
		{
			Name:    "update",
			Summary: "record progress on a local work item",
			Purpose: "Update a local work item summary and optionally change its next action or active status. Good updates explain what changed, what was learned, and what remains uncertain.",
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
				{Label: "Move into progress with context", Command: "aj update W-8F3K2P1Q --summary \"delete-path regression reproduced; root cause appears to be a skipped invalidation branch after soft-delete\" --status in_progress --next \"patch the delete branch and extend cache/service_test.go coverage\""},
			},
			Related:        []string{"show", "done", "ls"},
			Safety:         []string{"Use summary to explain the meaningful change or learning, not just that work happened. Update next whenever the best follow-up action changed."},
			WorkflowTags:   []string{"core", "progress"},
			SearchKeywords: []string{"progress", "status", "summary", "context", "authoring"},
		},
		{
			Name:    "block",
			Summary: "mark a local work item blocked",
			Purpose: "Move a local work item into blocked status and optionally attach a dependency that explains what must finish first. Blocking updates should explain why progress stopped and what condition unblocks it.",
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
				{Label: "Block on another item with context", Command: "aj block W-8F3K2P1Q --on W-2M9A1C7L --summary \"waiting on schema decision because the cache key format depends on the new tenant column\" --next \"resume once W-2M9A1C7L lands and rerun the regression suite\""},
			},
			Related:        []string{"unblock", "link", "show"},
			Safety:         []string{"Say why the blocker prevents progress and what evidence or dependency resolution will unblock the work."},
			WorkflowTags:   []string{"coordination", "blocked"},
			SearchKeywords: []string{"blocked", "wait", "dependency", "context", "authoring"},
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
			Purpose: "Mark a local work item done, record its completion summary, and clear its next action. Completion summaries should say what shipped or was verified so later agents do not need to rediscover the outcome.",
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
				{Label: "Complete an item with context", Command: "aj done W-8F3K2P1Q --summary \"delete-path invalidation fixed, regression coverage added in cache/service_test.go, and manual smoke test passed\""},
			},
			Related:        []string{"update", "show", "ls"},
			Safety:         []string{"Summaries should capture the shipped behavior or verification evidence, not just that the item is finished."},
			WorkflowTags:   []string{"core", "complete"},
			SearchKeywords: []string{"complete", "finish", "close", "context", "authoring"},
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
			Purpose: "Assign a new lease owner with a handoff summary so another agent can continue the work without ambiguity. Handoff summaries should explain what is done, what is risky, and what to check next.",
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
				{Label: "Hand off for review with context", Command: "aj handoff W-8F3K2P1Q --to reviewer-1 --summary \"delete-path fix is in; main risk is multi-tenant cache key compatibility during upgrade\" --next \"verify the regression tests and spot-check upgrade-path behavior\""},
			},
			Related:        []string{"take", "release", "show"},
			Safety:         []string{"Use handoff when ownership is explicitly changing, rather than force-claiming another agent's active lease.", "A good handoff should let the receiving agent continue without guessing what was finished, what is risky, or what evidence still needs review."},
			WorkflowTags:   []string{"coordination", "handoff"},
			SearchKeywords: []string{"handoff", "transfer", "lease", "context", "authoring"},
		},
		{
			Name:    "checkpoint",
			Summary: "record a compact resume point without transferring ownership",
			Purpose: "Capture what changed, what remains risky, and what the next agent should verify so work can resume cleanly later even if ownership does not change yet.",
			Usage:   "aj checkpoint <id> --summary <summary> [--next <action>] [--risk <text> ...] [--verify <text> ...]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required compact checkpoint summary"},
				{Name: "--next <action>", Description: "optional replacement next action for whoever resumes the work"},
				{Name: "--risk <text>", Description: "optional remaining risk or uncertainty; repeat as needed"},
				{Name: "--verify <text>", Description: "optional thing the next agent should verify; repeat as needed"},
			},
			Examples: []ExampleDoc{
				{Label: "Record a checkpoint", Command: "aj checkpoint W-8F3K2P1Q --summary \"export path is fixed; remaining risk is Jira projects with nonstandard transition names\" --next \"smoke-test against SD and inspect transitions output\" --risk \"mapped status names may not exist on all boards\" --verify \"run aj jira transitions <id> before syncing\""},
			},
			Related:        []string{"handoff", "show", "update"},
			Safety:         []string{"Use checkpoints when another agent may need to resume later, even if you are not transferring the lease yet.", "A good checkpoint should say what changed, the main remaining risks, and what to verify next."},
			WorkflowTags:   []string{"coordination", "handoff", "checkpoint"},
			SearchKeywords: []string{"checkpoint", "resume", "handoff", "continue", "risks", "verify"},
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
			Summary: "link a work item to another by dependency or hierarchy",
			Purpose: "Record either a dependency edge or a parent-child hierarchy edge so aj can understand readiness and work decomposition.",
			Usage:   "aj link <id> (--depends-on <id> | --parent <id>)",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier to update", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--depends-on <id>", Description: "record that the item cannot start until the dependency is done"},
				{Name: "--parent <id>", Description: "record that the item is a child of the given parent or epic item"},
			},
			Examples: []ExampleDoc{
				{Label: "Link a dependency", Command: "aj link W-CHILD --depends-on W-PARENT"},
				{Label: "Attach a child item to a parent", Command: "aj link W-CHILD --parent W-EPIC"},
			},
			Related:        []string{"unlink", "show", "next", "inbox"},
			Safety:         []string{"Use --depends-on for readiness constraints and --parent for decomposition; they serve different coordination purposes.", "aj rejects self-links and simple relation cycles so the graph stays actionable."},
			WorkflowTags:   []string{"coordination", "dependency", "hierarchy"},
			SearchKeywords: []string{"depends_on", "dependency", "waiting", "parent", "child", "epic"},
		},
		{
			Name:    "unlink",
			Summary: "remove a local dependency or parent relation",
			Purpose: "Remove one dependency edge or clear a parent link when the work graph changes and agents need the current structure to stay accurate.",
			Usage:   "aj unlink <id> (--depends-on <id> | --parent)",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier to update", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--depends-on <id>", Description: "remove one dependency edge from the item"},
				{Name: "--parent", Description: "clear the current parent link from the item"},
			},
			Examples: []ExampleDoc{
				{Label: "Remove a dependency", Command: "aj unlink W-CHILD --depends-on W-PARENT"},
				{Label: "Remove a parent link", Command: "aj unlink W-CHILD --parent"},
			},
			Related:        []string{"link", "show"},
			WorkflowTags:   []string{"coordination", "dependency", "hierarchy"},
			SearchKeywords: []string{"unlink", "dependency", "parent", "child", "graph"},
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
			Name:    "attach",
			Summary: "attach a durable file artifact to a work item",
			Purpose: "Copy a supporting file such as a log, patch, screenshot, or note into .aj/artifacts and record a compact summary so another agent can inspect the evidence later.",
			Usage:   "aj attach <id> --path <path> --summary <summary> [--label <label>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--path <path>", Description: "required file path to copy into .aj/artifacts"},
				{Name: "--summary <summary>", Description: "required compact explanation of why the artifact matters"},
				{Name: "--label <label>", Description: "optional short label for the artifact"},
			},
			Examples: []ExampleDoc{
				{Label: "Attach a failing test log", Command: "aj attach W-8F3K2P1Q --path /tmp/go-test.log --summary \"failing regression output before the cache fix\" --label pre-fix-log"},
			},
			Related:        []string{"receipt", "artifacts", "show"},
			Safety:         []string{"Summaries should explain what the file proves or why another agent should inspect it; avoid attaching huge files unless they materially help verification."},
			WorkflowTags:   []string{"evidence", "artifacts"},
			SearchKeywords: []string{"artifact", "attach", "log", "evidence", "receipt"},
		},
		{
			Name:    "receipt",
			Summary: "record a compact execution receipt for a work item",
			Purpose: "Record a command, exit status, and optional output file as durable evidence that a test, build, or verification step was attempted.",
			Usage:   "aj receipt <id> --summary <summary> --command <command> --exit-code <code> [--output <path>] [--label <label>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--summary <summary>", Description: "required compact explanation of the command result"},
				{Name: "--command <command>", Description: "required command that was run"},
				{Name: "--exit-code <code>", Description: "required integer exit code from the command"},
				{Name: "--output <path>", Description: "optional output file to copy into .aj/artifacts"},
				{Name: "--label <label>", Description: "optional short label for the receipt"},
			},
			Examples: []ExampleDoc{
				{Label: "Record a test run", Command: "aj receipt W-8F3K2P1Q --summary \"go test failed in cache package before the fix\" --command \"go test ./...\" --exit-code 1 --output /tmp/go-test.log --label test-failure"},
			},
			Related:        []string{"attach", "artifacts", "show"},
			Safety:         []string{"Use receipts for compact build or test evidence, not full shell transcripts; the summary should say what another agent should conclude from the command result."},
			WorkflowTags:   []string{"evidence", "artifacts"},
			SearchKeywords: []string{"receipt", "test", "command", "artifact", "evidence"},
		},
		{
			Name:    "artifacts",
			Summary: "list artifacts attached to one work item",
			Purpose: "Show the durable artifacts and execution receipts recorded for a work item so another agent can inspect evidence quickly.",
			Usage:   "aj artifacts <id> [--limit <n>]",
			Arguments: []ArgDoc{
				{Name: "id", Description: "required work item identifier such as W-8F3K2P1Q", Required: true},
			},
			Options: []OptionDoc{
				{Name: "--limit <n>", Description: "optional max number of artifacts to return; default 20"},
			},
			Examples: []ExampleDoc{
				{Label: "List one item's artifacts", Command: "aj artifacts W-8F3K2P1Q --limit 10"},
			},
			Related:        []string{"attach", "receipt", "show"},
			WorkflowTags:   []string{"evidence", "artifacts"},
			SearchKeywords: []string{"artifact", "evidence", "receipts", "logs"},
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
			Name:    "report",
			Summary: "show a compact repo-level work summary",
			Purpose: "Summarize the current local backlog with status counts plus the most relevant owned, ready, waiting, and recent work so an agent can orient quickly.",
			Usage:   "aj report [--agent <name>] [--limit <n>]",
			Options: []OptionDoc{
				{Name: "--agent <name>", Description: "optional agent name; if provided, owned and ready sections are tailored to that agent"},
				{Name: "--limit <n>", Description: "optional max entries per report section; default 5"},
			},
			Examples: []ExampleDoc{
				{Label: "Show a repo summary", Command: "aj report"},
				{Label: "Show a report for one agent", Command: "aj report --agent coder-1 --limit 3"},
			},
			Related:        []string{"search", "next", "inbox", "ready", "changes"},
			WorkflowTags:   []string{"discovery", "query", "reporting"},
			SearchKeywords: []string{"report", "summary", "dashboard", "counts"},
		},
		{
			Name:    "jira",
			Summary: "import or export Jira issues",
			Purpose: "Use the Jira adapter to search for likely existing issues, pull Jira issues into local aj items, push local items to Jira, manage Jira spaces, link or unlink local items, inspect status mappings and transitions, sync linked items with conflict detection, and send compact milestone comments back to Jira.",
			Usage:   "aj jira <search|pull|push|space|link|unlink|status-map|transitions|sync|comment> ...",
			Options: []OptionDoc{
				{Name: "search <terms...> [--limit <n>] [--project <key>]", Description: "search Jira for existing issues, defaulting to the configured project when one is set"},
				{Name: "pull <key>", Description: "import a Jira issue into aj unless it is already linked"},
				{Name: "push <id>", Description: "create a Jira issue from a local aj item unless it is already linked"},
				{Name: "space exists [--key <key>]", Description: "check whether a Jira space/project key already exists; defaults to the configured Jira project key when available"},
				{Name: "space create --key <key> --name <name> [--type <type>] [--template <template>]", Description: "create a new Jira space/project explicitly"},
				{Name: "space ensure --key <key> --name <name> [--type <type>] [--template <template>]", Description: "return the existing Jira space/project for a key or create it if missing"},
				{Name: "space ls [--query <text>] [--limit <n>]", Description: "list visible Jira spaces/projects, optionally filtered by a search query"},
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
				{Label: "Search for likely existing work", Command: "aj jira search cache invalidation"},
				{Label: "Check whether a Jira space key exists", Command: "aj jira space exists --key SD"},
				{Label: "Create a Jira space", Command: "aj jira space create --key SD --name \"Software Delivery\""},
				{Label: "Ensure a Jira space exists", Command: "aj jira space ensure --key SD --name \"Software Delivery\""},
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
			Safety:         []string{"Set Jira credentials through environment variables instead of committing secrets into .aj/config.toml.", "Use `aj jira search ...` before creating or linking new work so agents can avoid duplicate issues.", "When bootstrapping a repo, decide whether the Jira space key should point at an existing space or whether `aj` should create one with `aj init --ensure-jira-space ...` or `aj jira space ensure ...`.", "Write Jira-facing descriptions and milestone comments so a different agent can understand why the work exists, what changed, and what still needs verification.", "Use `aj jira status-map` and `aj jira transitions <id>` before the first sync or whenever local and remote workflow behavior is unclear.", "Use `aj jira unlink --force` only when you intentionally want to drop a dirty or conflicted Jira link.", "Sync may attempt a Jira status transition when local status differs and a mapped transition exists."},
			WorkflowTags:   []string{"jira", "integration", "spaces"},
			SearchKeywords: []string{"jira", "search", "find", "import", "export", "sync", "comment", "transition", "status map", "workflow", "unlink", "relink", "space", "project", "ensure"},
		},
		{
			Name:    "init",
			Summary: "create the .aj workspace structure in the current repository",
			Purpose: "Bootstrap a repository for aj by creating the .aj directory tree, default config, cache directory, and artifact directories. When Jira is part of the workflow, init can also record the Jira site and space key, and optionally ensure that the Jira space exists before work starts.",
			Usage:   "aj init [--repo <path>] [--force] [--jira --jira-base-url <url> --jira-space-key <key> [--jira-space-name <name>] [--jira-space-type <type>] [--jira-space-template <template>] [--ensure-jira-space]]",
			Options: []OptionDoc{
				{Name: "--repo <path>", Description: "initialize aj in a specific repository path instead of the current working directory"},
				{Name: "--force", Description: "overwrite an existing config file if present"},
				{Name: "--jira", Description: "enable Jira support in the new .aj/config.toml"},
				{Name: "--jira-base-url <url>", Description: "set the Jira Cloud base URL such as https://example.atlassian.net"},
				{Name: "--jira-space-key <key>", Description: "set the Jira space/project key to reuse or create"},
				{Name: "--jira-space-name <name>", Description: "human-readable Jira space/project name to use when creating a missing space"},
				{Name: "--jira-space-type <type>", Description: "optional Jira project type such as software; default software when ensuring a space"},
				{Name: "--jira-space-template <template>", Description: "optional Jira project template key; required for some non-software project types"},
				{Name: "--ensure-jira-space", Description: "check whether the Jira space key exists and create it if missing"},
				{Name: "--format brief|json|prompt", Description: "render the result in the requested output format"},
			},
			Examples: []ExampleDoc{
				{Label: "Initialize current repo", Command: "aj init"},
				{Label: "Initialize another repo", Command: "aj --repo /path/to/repo init"},
				{Label: "Initialize with an existing Jira space key", Command: "aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD"},
				{Label: "Initialize and create the Jira space if missing", Command: "aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD --jira-space-name \"Software Delivery\" --ensure-jira-space"},
			},
			Related:        []string{"help", "commands"},
			Safety:         []string{"If Jira is enabled during init, decide whether the configured space key should reuse an existing Jira space or whether aj should create it when missing.", "Use `--ensure-jira-space` only when the configured Jira credentials have permission to create spaces/projects in that Jira site."},
			WorkflowTags:   []string{"core", "jira", "bootstrap"},
			SearchKeywords: []string{"bootstrap", "setup", "repository", "jira", "space", "project key", "create"},
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
				"2. If Jira will be used, decide whether the Jira space key already exists or whether aj should create it during bootstrap.",
				"3. Use `aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD` to reuse an existing Jira space, or add `--jira-space-name \"Software Delivery\" --ensure-jira-space` to create it when missing.",
				"4. Run `aj new --kind task --title \"...\" --goal \"...\" --next \"...\"` to create the first item.",
				"5. Use `aj ls` to inspect the local queue.",
				"6. Use `aj show <id>` before updating or handing work off.",
			},
		},
		"jira": {
			Name:  "jira",
			Topic: "bootstrap Jira-aware repos and search, import, or claim human-created Jira work",
			Steps: []string{
				"1. Set `AJ_JIRA_EMAIL` and `AJ_JIRA_API_TOKEN` for the Jira site you want aj to use.",
				"2. Decide whether the Jira space key already exists. Use `aj jira space exists --key SD` or `aj jira space ls --query delivery` to inspect what's already there.",
				"3. Bootstrap the repo with `aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD`, or add `--jira-space-name \"Software Delivery\" --ensure-jira-space` when the space should be created if missing.",
				"4. Run `aj jira search ...` to look for likely existing Jira work before creating or linking anything new.",
				"5. Run `aj take jira ABC-123 --agent coder-1` to import and claim human-created work.",
				"6. Use `aj show <id>` to inspect the normalized local item.",
				"7. Use `aj jira status-map` and `aj jira transitions <id>` to inspect the workflow before syncing.",
				"8. Use `aj update <id> --summary ... --next ...` to record progress, `aj jira comment <id> --summary \"ready for review\"` for milestone updates, or `aj jira sync <id> --dry-run` to preview sync direction and status transitions.",
				"9. Use `aj jira unlink <id>` before relinking to a different Jira issue, or `aj jira link <id> <key> --replace` when that swap is intentional.",
			},
		},
		"authoring": {
			Name:  "authoring",
			Topic: "write tickets and updates with enough context for another agent to continue the work",
			Steps: []string{
				"1. Make the title describe the concrete problem or outcome, not just an activity like \"work on sync\".",
				"2. Use the goal to explain why the work matters, then add structured fields for acceptance criteria, constraints, risks, relevant files, and verification whenever they would help another agent execute accurately.",
				"3. Use progress, block, handoff, and done summaries to explain what changed, what was learned, and what risk or uncertainty remains.",
				"4. Make next actions concrete enough that another agent can start from them without rereading the whole codebase.",
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
		"checkpoint": {
			Name:  "checkpoint",
			Topic: "leave a compact resume point before pausing or handing work off",
			Steps: []string{
				"1. Use `aj checkpoint <id> --summary \"...\"` when you have meaningful state another agent should inherit later.",
				"2. Add `--risk \"...\"` entries for the main uncertainties or sharp edges that remain.",
				"3. Add `--verify \"...\"` entries for the checks the next agent should run or inspect.",
				"4. Add `--next \"...\"` when the best next action changed and should be explicit in the snapshot.",
				"5. Use `aj handoff ...` after the checkpoint when ownership is actually changing.",
			},
		},
		"evidence": {
			Name:  "evidence",
			Topic: "attach proof so another agent can verify what was tried and what happened",
			Steps: []string{
				"1. Use `aj receipt <id> --summary \"...\" --command \"...\" --exit-code <n>` after an important build, test, or verification command.",
				"2. Add `--output /path/to/log` when the command produced a log another agent may need to inspect.",
				"3. Use `aj attach <id> --path /path/to/file --summary \"...\"` for patches, screenshots, notes, or other supporting evidence.",
				"4. Use `aj show <id>` or `aj artifacts <id>` to confirm the evidence is attached with a useful summary.",
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
				"1. Use `aj checkpoint <id> --summary \"...\" --risk \"...\" --verify \"...\"` first when the current state needs a stronger resume point.",
				"2. Use `aj show <id>` before handing work off so the current snapshot is accurate.",
				"3. Use `aj handoff <id> --to reviewer-1 --summary \"...\" --next \"...\"` to transfer the lease.",
				"4. Use `aj inbox --agent reviewer-1` or `aj next --agent reviewer-1` so the receiving agent can pick it up immediately.",
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
			Topic: "declare dependency and hierarchy edges so waiting work is not recommended too early and child work stays grouped",
			Steps: []string{
				"1. Use `aj link <id> --depends-on <dependency>` when work cannot start until another item is done.",
				"2. Use `aj link <id> --parent <parent>` when the item should roll up under a larger task or epic.",
				"3. Use `aj show <id>` to confirm parent, children, dependencies, and blocked downstream work are recorded the way you expect.",
				"4. Use `aj unlink ...` when plans change so the graph stays accurate.",
				"5. Use `aj inbox --agent coder-1` or `aj next --agent coder-1` and let aj mark dependent work as waiting.",
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
		"reporting": {
			Name:  "reporting",
			Topic: "query the local backlog without manually scanning every item",
			Steps: []string{
				"1. Use `aj search ...` to find related existing work before creating a new ticket or updating an old one.",
				"2. Add `--status` or `--kind` filters when you want a tighter slice like blocked bugs or in-progress features.",
				"3. Use `aj report` to get a compact repo summary with status counts plus owned, ready, waiting, and recent sections.",
				"4. Use `aj report --agent coder-1` when an individual agent needs a fast orientation pass before taking new work.",
			},
		},
	}

	examples := map[string]ExampleSet{
		"init": {
			Topic: "init",
			Examples: []ExampleDoc{
				{Label: "Initialize current repository", Command: "aj init"},
				{Label: "Initialize a repo from another working directory", Command: "aj --repo /workspace/project init"},
				{Label: "Reuse an existing Jira space during init", Command: "aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD"},
				{Label: "Create the Jira space during init if needed", Command: "aj init --jira --jira-base-url https://example.atlassian.net --jira-space-key SD --jira-space-name \"Software Delivery\" --ensure-jira-space"},
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
		"checkpoint": {
			Topic: "checkpoint",
			Examples: []ExampleDoc{
				{Label: "Leave a resume point", Command: "aj checkpoint W-8F3K2P1Q --summary \"status alignment is fixed; remaining risk is Jira projects without the mapped transition\" --next \"smoke-test a second project\" --risk \"transition names may differ by workflow\" --verify \"run aj jira transitions <id> before sync\""},
			},
		},
		"queue": {
			Topic: "queue",
			Examples: []ExampleDoc{
				{Label: "Find one next item", Command: "aj next --agent coder-1"},
				{Label: "Show the current inbox", Command: "aj inbox --agent coder-1"},
			},
		},
		"reporting": {
			Topic: "reporting",
			Examples: []ExampleDoc{
				{Label: "Search local work", Command: "aj search cache invalidation"},
				{Label: "Find blocked bugs", Command: "aj search regression --status blocked --kind bug"},
				{Label: "Summarize the repo", Command: "aj report"},
				{Label: "Summarize one agent's queue", Command: "aj report --agent coder-1 --limit 3"},
			},
		},
		"dependencies": {
			Topic: "dependencies",
			Examples: []ExampleDoc{
				{Label: "Record a dependency", Command: "aj link W-CHILD --depends-on W-PARENT"},
				{Label: "Attach a child to an epic", Command: "aj link W-CHILD --parent W-EPIC"},
				{Label: "Remove an outdated dependency", Command: "aj unlink W-CHILD --depends-on W-PARENT"},
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
				{Label: "Check whether a Jira space exists", Command: "aj jira space exists --key SD"},
				{Label: "List visible Jira spaces", Command: "aj jira space ls --query delivery"},
				{Label: "Create a Jira space", Command: "aj jira space create --key SD --name \"Software Delivery\""},
				{Label: "Ensure a Jira space exists", Command: "aj jira space ensure --key SD --name \"Software Delivery\""},
				{Label: "Search Jira for existing work", Command: "aj jira search cache invalidation"},
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
		"authoring": {
			Topic: "authoring",
			Examples: []ExampleDoc{
				{Label: "Create a context-rich item", Command: "aj new --kind feature --title \"Align Jira status on export\" --goal \"ensure jira push lands exported issues in the correct Jira workflow state and keep existing export-only links repairable\" --next \"patch export to transition after create and add regression coverage\" --accept \"newly exported issues land in the mapped Jira status\" --accept \"older export-only links can be repaired with sync\" --constraint \"do not regress import-only links\" --risk \"some Jira projects may not expose the desired transition name\" --file internal/store/item.go --file internal/jira/client.go --verify \"run go test ./... and smoke-test against SD\""},
				{Label: "Write a useful progress update", Command: "aj update W-8F3K2P1Q --summary \"export now records remote versions, but old export-only links still need a repair path\" --next \"treat empty last_remote_version as dirty_local and verify with a live sync\""},
				{Label: "Leave a reusable checkpoint", Command: "aj checkpoint W-8F3K2P1Q --summary \"status alignment is fixed; remaining risk is Jira projects without the mapped transition\" --next \"smoke-test a second project\" --risk \"transition names may differ by workflow\" --verify \"run aj jira transitions <id> before sync\""},
				{Label: "Write a handoff another agent can use", Command: "aj handoff W-8F3K2P1Q --to reviewer-1 --summary \"status alignment is fixed; main remaining risk is Jira projects that lack the mapped transition names\" --next \"verify the live SD board and inspect transition diagnostics output\""},
			},
		},
		"evidence": {
			Topic: "evidence",
			Examples: []ExampleDoc{
				{Label: "Record a failing test receipt", Command: "aj receipt W-8F3K2P1Q --summary \"cache regression still fails before the fix\" --command \"go test ./...\" --exit-code 1 --output /tmp/go-test.log --label pre-fix-test"},
				{Label: "Attach a patch for review", Command: "aj attach W-8F3K2P1Q --path /tmp/cache-fix.patch --summary \"candidate patch for cache invalidation review\" --label patch"},
				{Label: "Inspect attached evidence", Command: "aj artifacts W-8F3K2P1Q --limit 10"},
			},
		},
	}

	glossary := map[string]GlossaryEntry{
		"context-rich update": {
			Term:       "context-rich update",
			Definition: "A summary, goal, or handoff note that explains the relevant why, what changed, and what comes next so another agent can continue without rediscovering the same context.",
		},
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
