package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"ajentwork/internal/app"
	"ajentwork/internal/config"
	"ajentwork/internal/domain"
	"ajentwork/internal/help"
	"ajentwork/internal/render"
)

type Runner struct {
	stdout io.Writer
	stderr io.Writer
	help   help.Registry
}

func NewRunner(stdout, stderr io.Writer) Runner {
	return Runner{
		stdout: stdout,
		stderr: stderr,
		help:   help.DefaultRegistry(),
	}
}

type globalOptions struct {
	repoPath string
	format   domain.OutputFormat
}

func (r Runner) Run(args []string) int {
	exitCode, err := r.run(args)
	if err == nil {
		return exitCode
	}

	fmt.Fprintln(r.stderr, err.Error())
	return exitCode
}

func (r Runner) run(args []string) (int, error) {
	globals, remaining, err := parseGlobalOptions(args)
	if err != nil {
		return 2, err
	}

	if globals.repoPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return 1, fmt.Errorf("resolve working directory: %w", err)
		}
		globals.repoPath = cwd
	}

	if len(remaining) == 0 {
		return r.renderRootHelp(globals.format)
	}

	if remaining[0] == "--help" || remaining[0] == "-h" {
		return r.renderRootHelp(globals.format)
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	switch command {
	case "init":
		return r.runInit(globals, commandArgs)
	case "new":
		return r.runNew(globals, commandArgs)
	case "ls":
		return r.runList(globals, commandArgs)
	case "show":
		return r.runShow(globals, commandArgs)
	case "update":
		return r.runUpdate(globals, commandArgs)
	case "block":
		return r.runBlock(globals, commandArgs)
	case "unblock":
		return r.runUnblock(globals, commandArgs)
	case "done":
		return r.runDone(globals, commandArgs)
	case "take":
		return r.runTake(globals, commandArgs)
	case "release":
		return r.runRelease(globals, commandArgs)
	case "handoff":
		return r.runHandoff(globals, commandArgs)
	case "reopen":
		return r.runReopen(globals, commandArgs)
	case "next":
		return r.runNext(globals, commandArgs)
	case "inbox":
		return r.runInbox(globals, commandArgs)
	case "link":
		return r.runLink(globals, commandArgs)
	case "changes":
		return r.runChanges(globals, commandArgs)
	case "ready":
		return r.runReady(globals, commandArgs)
	case "jira":
		return r.runJira(globals, commandArgs)
	case "help":
		return r.runHelp(globals, commandArgs)
	case "commands":
		return r.runCommands(globals, commandArgs)
	case "workflows":
		return r.runWorkflows(globals, commandArgs)
	case "examples":
		return r.runExamples(globals, commandArgs)
	case "glossary":
		return r.runGlossary(globals, commandArgs)
	default:
		return 2, fmt.Errorf("unknown command %q\ntry: aj help", command)
	}
}

func parseGlobalOptions(args []string) (globalOptions, []string, error) {
	options := globalOptions{format: domain.FormatBrief}
	remaining := make([]string, 0, len(args))

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--repo":
			i++
			if i >= len(args) {
				return globalOptions{}, nil, errors.New("missing value for --repo")
			}
			options.repoPath = args[i]
		case "--format":
			i++
			if i >= len(args) {
				return globalOptions{}, nil, errors.New("missing value for --format")
			}
			format, err := domain.ParseOutputFormat(args[i])
			if err != nil {
				return globalOptions{}, nil, err
			}
			options.format = format
		case "--help", "-h":
			remaining = append(remaining, arg)
		default:
			remaining = append(remaining, arg)
		}
	}

	return options, remaining, nil
}

func (r Runner) renderRootHelp(format domain.OutputFormat) (int, error) {
	doc := r.help.Root()
	commands := r.help.Commands()

	switch format {
	case domain.FormatBrief:
		_, err := fmt.Fprintln(r.stdout, render.RootHelp(doc, commands))
		return 0, err
	case domain.FormatPrompt:
		promptDoc := help.Doc{
			Name:    "aj",
			Purpose: doc.Purpose,
			Usage:   doc.Usage,
			Related: []string{"help", "commands", "workflows", "examples", "glossary"},
		}
		_, err := fmt.Fprintln(r.stdout, render.CommandHelpPrompt(promptDoc))
		return 0, err
	case domain.FormatJSON:
		payload := struct {
			Doc      help.Doc              `json:"doc"`
			Commands []help.CommandSummary `json:"commands"`
		}{Doc: doc, Commands: commands}
		return r.renderJSON(payload)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) runInit(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("init", globals.format)
		}
	}

	force := false
	for _, arg := range args {
		switch arg {
		case "--force":
			force = true
		default:
			return 2, fmt.Errorf("unknown option for init: %s", arg)
		}
	}

	service := app.InitService{}
	result, err := service.Run(globals.repoPath, force)
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if result.AlreadyReady {
			_, err = fmt.Fprintf(r.stdout, "aj is already initialized in %s\nconfig: %s\n", result.RepoPath, result.ConfigPath)
		} else {
			_, err = fmt.Fprintf(r.stdout, "initialized aj in %s\nconfig: %s\n", result.RepoPath, result.ConfigPath)
		}
		return 0, err
	case domain.FormatPrompt:
		status := "initialized"
		if result.AlreadyReady {
			status = "already initialized"
		}
		_, err = fmt.Fprintf(r.stdout, "Status: %s\nRepo: %s\nConfig: %s\n", status, result.RepoPath, result.ConfigPath)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runNew(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("new", globals.format)
		}
	}

	kindRaw := ""
	title := ""
	goal := ""
	nextAction := ""
	priority := 2

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--kind":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --kind")
			}
			kindRaw = args[i]
		case "--title":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --title")
			}
			title = args[i]
		case "--goal":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --goal")
			}
			goal = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			nextAction = args[i]
		case "--priority":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --priority")
			}
			value, err := strconv.Atoi(args[i])
			if err != nil {
				return 2, fmt.Errorf("invalid priority %q", args[i])
			}
			priority = value
		default:
			return 2, fmt.Errorf("unknown option for new: %s", args[i])
		}
	}

	kind, err := domain.ParseItemKind(kindRaw)
	if err != nil {
		return 2, err
	}

	service := app.NewItemService{}
	item, err := service.Run(app.NewItemInput{
		RepoPath:   globals.repoPath,
		Kind:       kind,
		Title:      title,
		Goal:       goal,
		NextAction: nextAction,
		Priority:   priority,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemCreatedBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemCreatedPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runList(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("ls", globals.format)
		}
	}
	if len(args) > 0 {
		return 2, errors.New("usage: aj ls")
	}

	service := app.ListItemsService{}
	items, err := service.Run(globals.repoPath)
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemListBrief(items))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemListPrompt(items))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(items)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runShow(globals globalOptions, args []string) (int, error) {
	showHistory := false
	historyLimit := 5
	itemID := ""

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--help", "-h":
			return r.renderCommandHelp("show", globals.format)
		case "--history":
			showHistory = true
		case "--limit":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --limit")
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed < 0 {
				return 2, fmt.Errorf("invalid --limit value %q", args[i])
			}
			historyLimit = parsed
		default:
			if strings.HasPrefix(args[i], "--") {
				return 2, fmt.Errorf("unknown option for show: %s", args[i])
			}
			if itemID != "" {
				return 2, errors.New("usage: aj show <id> [--history] [--limit <n>]")
			}
			itemID = args[i]
		}
	}
	if itemID == "" {
		return 2, errors.New("usage: aj show <id> [--history] [--limit <n>]")
	}

	service := app.ShowItemService{}
	item, err := service.Run(globals.repoPath, itemID)
	if err != nil {
		return 1, err
	}

	var events []domain.Event
	if showHistory {
		changeService := app.ChangesService{}
		events, err = changeService.Run(app.ChangesInput{
			RepoPath: globals.repoPath,
			ItemID:   itemID,
			Limit:    historyLimit,
		})
		if err != nil {
			return 1, err
		}
	}

	switch globals.format {
	case domain.FormatBrief:
		if showHistory {
			_, err = fmt.Fprintln(r.stdout, render.ItemWithHistoryBrief(item, events))
		} else {
			_, err = fmt.Fprintln(r.stdout, render.ItemShowBrief(item))
		}
		return 0, err
	case domain.FormatPrompt:
		if showHistory {
			_, err = fmt.Fprintln(r.stdout, render.ItemWithHistoryPrompt(item, events))
		} else {
			_, err = fmt.Fprintln(r.stdout, render.ItemShowPrompt(item))
		}
		return 0, err
	case domain.FormatJSON:
		if showHistory {
			payload := struct {
				Item    domain.Item    `json:"item"`
				History []domain.Event `json:"history"`
			}{
				Item:    item,
				History: events,
			}
			return r.renderJSON(payload)
		}
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runUpdate(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("update", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj update <id> --summary <summary> [--next <action>] [--status <status>]")
	}

	itemID := args[0]
	var summary string
	var nextAction *string
	var status *domain.Status

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			value := args[i]
			nextAction = &value
		case "--status":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --status")
			}
			parsed, err := domain.ParseStatus(args[i])
			if err != nil {
				return 2, err
			}
			status = &parsed
		default:
			return 2, fmt.Errorf("unknown option for update: %s", args[i])
		}
	}

	service := app.UpdateItemService{}
	item, err := service.Run(app.UpdateItemInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		Summary:    summary,
		NextAction: nextAction,
		Status:     status,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemUpdatedBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemUpdatedPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runDone(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("done", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj done <id> --summary <summary> [--jira-comment|--no-jira-comment]")
	}

	itemID := args[0]
	summary := ""
	postJiraComment := false
	jiraCommentPreferenceSet := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--jira-comment":
			if jiraCommentPreferenceSet && !postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = true
			jiraCommentPreferenceSet = true
		case "--no-jira-comment":
			if jiraCommentPreferenceSet && postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = false
			jiraCommentPreferenceSet = true
		default:
			return 2, fmt.Errorf("unknown option for done: %s", args[i])
		}
	}
	if !jiraCommentPreferenceSet {
		var err error
		postJiraComment, err = r.lifecycleJiraCommentDefault(globals.repoPath, "done")
		if err != nil {
			return 1, err
		}
	}
	if jiraCommentPreferenceSet && postJiraComment {
		if err := r.ensureItemLinkedToJira(globals.repoPath, itemID); err != nil {
			return 1, err
		}
	}

	service := app.CompleteItemService{}
	item, err := service.Run(app.CompleteItemInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		Summary:  summary,
	})
	if err != nil {
		return 1, err
	}
	jiraCommentPosted := false
	if postJiraComment {
		if jiraCommentPreferenceSet || item.Jira != nil {
			if _, err := r.postLifecycleJiraComment(globals.repoPath, item.ID, summary); err != nil {
				return 1, err
			}
			jiraCommentPosted = true
		}
	}

	switch globals.format {
	case domain.FormatBrief:
		output := render.ItemDoneBrief(item)
		if jiraCommentPosted {
			output += "\nposted Jira milestone comment"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatPrompt:
		output := render.ItemDonePrompt(item)
		if jiraCommentPosted {
			output += "\nJira Comment: posted"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatJSON:
		if jiraCommentPosted {
			return r.renderJSON(struct {
				Item              domain.Item `json:"item"`
				JiraCommentPosted bool        `json:"jira_comment_posted"`
			}{Item: item, JiraCommentPosted: true})
		}
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runBlock(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("block", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj block <id> --summary <summary> [--on <id>] [--next <action>] [--jira-comment|--no-jira-comment]")
	}

	itemID := args[0]
	summary := ""
	onID := ""
	var nextAction *string
	postJiraComment := false
	jiraCommentPreferenceSet := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--on":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --on")
			}
			onID = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			value := args[i]
			nextAction = &value
		case "--jira-comment":
			if jiraCommentPreferenceSet && !postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = true
			jiraCommentPreferenceSet = true
		case "--no-jira-comment":
			if jiraCommentPreferenceSet && postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = false
			jiraCommentPreferenceSet = true
		default:
			return 2, fmt.Errorf("unknown option for block: %s", args[i])
		}
	}
	if !jiraCommentPreferenceSet {
		var err error
		postJiraComment, err = r.lifecycleJiraCommentDefault(globals.repoPath, "block")
		if err != nil {
			return 1, err
		}
	}
	if jiraCommentPreferenceSet && postJiraComment {
		if err := r.ensureItemLinkedToJira(globals.repoPath, itemID); err != nil {
			return 1, err
		}
	}

	service := app.BlockItemService{}
	item, err := service.Run(app.BlockItemInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		Summary:    summary,
		OnID:       onID,
		NextAction: nextAction,
	})
	if err != nil {
		return 1, err
	}
	jiraCommentPosted := false
	if postJiraComment {
		if jiraCommentPreferenceSet || item.Jira != nil {
			if _, err := r.postLifecycleJiraComment(globals.repoPath, item.ID, summary); err != nil {
				return 1, err
			}
			jiraCommentPosted = true
		}
	}

	switch globals.format {
	case domain.FormatBrief:
		output := render.ItemBlockedBrief(item)
		if jiraCommentPosted {
			output += "\nposted Jira milestone comment"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatPrompt:
		output := render.ItemBlockedPrompt(item)
		if jiraCommentPosted {
			output += "\nJira Comment: posted"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatJSON:
		if jiraCommentPosted {
			return r.renderJSON(struct {
				Item              domain.Item `json:"item"`
				JiraCommentPosted bool        `json:"jira_comment_posted"`
			}{Item: item, JiraCommentPosted: true})
		}
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runUnblock(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("unblock", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj unblock <id> --summary <summary> [--next <action>] [--status <status>]")
	}

	itemID := args[0]
	summary := ""
	var nextAction *string
	var status *domain.Status

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			value := args[i]
			nextAction = &value
		case "--status":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --status")
			}
			parsed, err := domain.ParseStatus(args[i])
			if err != nil {
				return 2, err
			}
			status = &parsed
		default:
			return 2, fmt.Errorf("unknown option for unblock: %s", args[i])
		}
	}

	service := app.UnblockItemService{}
	item, err := service.Run(app.UnblockItemInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		Summary:    summary,
		NextAction: nextAction,
		Status:     status,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemUnblockedBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemUnblockedPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runTake(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("take", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj take <id> --agent <name> [--ttl 4h] [--force]")
	}
	if args[0] == "jira" {
		return r.runTakeJira(globals, args[1:])
	}

	itemID := args[0]
	agent := ""
	ttl := 4 * time.Hour
	force := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --agent")
			}
			agent = args[i]
		case "--ttl":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --ttl")
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil {
				return 2, fmt.Errorf("invalid ttl %q", args[i])
			}
			ttl = parsed
		case "--force":
			force = true
		default:
			return 2, fmt.Errorf("unknown option for take: %s", args[i])
		}
	}

	service := app.TakeItemService{}
	item, err := service.Run(app.TakeItemInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		Agent:    agent,
		TTL:      ttl,
		Force:    force,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemTakenBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemTakenPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runTakeJira(globals globalOptions, args []string) (int, error) {
	if len(args) == 0 {
		return 2, errors.New("usage: aj take jira <key> --agent <name> [--ttl 4h] [--force]")
	}

	issueKey := args[0]
	agent := ""
	ttl := 4 * time.Hour
	force := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --agent")
			}
			agent = args[i]
		case "--ttl":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --ttl")
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil {
				return 2, fmt.Errorf("invalid ttl %q", args[i])
			}
			ttl = parsed
		case "--force":
			force = true
		default:
			return 2, fmt.Errorf("unknown option for take jira: %s", args[i])
		}
	}

	importService := app.ImportJiraIssueService{}
	imported, err := importService.Run(app.ImportJiraIssueInput{
		RepoPath: globals.repoPath,
		IssueKey: issueKey,
	})
	if err != nil {
		return 1, err
	}

	takeService := app.TakeItemService{}
	item, err := takeService.Run(app.TakeItemInput{
		RepoPath: globals.repoPath,
		ItemID:   imported.Item.ID,
		Agent:    agent,
		TTL:      ttl,
		Force:    force,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if imported.AlreadyLinked {
			_, err = fmt.Fprintf(r.stdout, "using existing %s for Jira %s\n%s\n", item.ID, issueKey, render.ItemTakenBrief(item))
		} else {
			_, err = fmt.Fprintf(r.stdout, "imported Jira %s as %s\n%s\n", issueKey, item.ID, render.ItemTakenBrief(item))
		}
		return 0, err
	case domain.FormatPrompt:
		if imported.AlreadyLinked {
			_, err = fmt.Fprintf(r.stdout, "Status: existing Jira link\nJira: %s\nID: %s\n%s\n", issueKey, item.ID, render.ItemTakenPrompt(item))
		} else {
			_, err = fmt.Fprintf(r.stdout, "Status: imported Jira issue\nJira: %s\nID: %s\n%s\n", issueKey, item.ID, render.ItemTakenPrompt(item))
		}
		return 0, err
	case domain.FormatJSON:
		payload := struct {
			Item          domain.Item `json:"item"`
			AlreadyLinked bool        `json:"already_linked"`
		}{
			Item:          item,
			AlreadyLinked: imported.AlreadyLinked,
		}
		return r.renderJSON(payload)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runRelease(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("release", globals.format)
		}
	}
	if len(args) != 1 {
		return 2, errors.New("usage: aj release <id>")
	}

	service := app.ReleaseItemService{}
	item, err := service.Run(app.ReleaseItemInput{
		RepoPath: globals.repoPath,
		ItemID:   args[0],
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemReleasedBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemReleasedPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runHandoff(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("handoff", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj handoff <id> --to <agent> --summary <summary> [--next <action>] [--ttl 4h] [--jira-comment|--no-jira-comment]")
	}

	itemID := args[0]
	toAgent := ""
	summary := ""
	var nextAction *string
	ttl := 4 * time.Hour
	postJiraComment := false
	jiraCommentPreferenceSet := false

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--to":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --to")
			}
			toAgent = args[i]
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			value := args[i]
			nextAction = &value
		case "--ttl":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --ttl")
			}
			parsed, err := time.ParseDuration(args[i])
			if err != nil {
				return 2, fmt.Errorf("invalid ttl %q", args[i])
			}
			ttl = parsed
		case "--jira-comment":
			if jiraCommentPreferenceSet && !postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = true
			jiraCommentPreferenceSet = true
		case "--no-jira-comment":
			if jiraCommentPreferenceSet && postJiraComment {
				return 2, errors.New("cannot combine --jira-comment and --no-jira-comment")
			}
			postJiraComment = false
			jiraCommentPreferenceSet = true
		default:
			return 2, fmt.Errorf("unknown option for handoff: %s", args[i])
		}
	}
	if !jiraCommentPreferenceSet {
		var err error
		postJiraComment, err = r.lifecycleJiraCommentDefault(globals.repoPath, "handoff")
		if err != nil {
			return 1, err
		}
	}
	if jiraCommentPreferenceSet && postJiraComment {
		if err := r.ensureItemLinkedToJira(globals.repoPath, itemID); err != nil {
			return 1, err
		}
	}

	service := app.HandoffItemService{}
	item, err := service.Run(app.HandoffItemInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		ToAgent:    toAgent,
		Summary:    summary,
		NextAction: nextAction,
		TTL:        ttl,
	})
	if err != nil {
		return 1, err
	}
	jiraCommentPosted := false
	if postJiraComment {
		if jiraCommentPreferenceSet || item.Jira != nil {
			if _, err := r.postLifecycleJiraComment(globals.repoPath, item.ID, summary); err != nil {
				return 1, err
			}
			jiraCommentPosted = true
		}
	}

	switch globals.format {
	case domain.FormatBrief:
		output := render.ItemHandedOffBrief(item)
		if jiraCommentPosted {
			output += "\nposted Jira milestone comment"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatPrompt:
		output := render.ItemHandedOffPrompt(item)
		if jiraCommentPosted {
			output += "\nJira Comment: posted"
		}
		_, err = fmt.Fprintln(r.stdout, output)
		return 0, err
	case domain.FormatJSON:
		if jiraCommentPosted {
			return r.renderJSON(struct {
				Item              domain.Item `json:"item"`
				JiraCommentPosted bool        `json:"jira_comment_posted"`
			}{Item: item, JiraCommentPosted: true})
		}
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runReopen(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("reopen", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj reopen <id> --summary <summary> --next <action> [--status <status>]")
	}

	itemID := args[0]
	summary := ""
	nextAction := ""
	var status *domain.Status

	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		case "--next":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --next")
			}
			nextAction = args[i]
		case "--status":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --status")
			}
			parsed, err := domain.ParseStatus(args[i])
			if err != nil {
				return 2, err
			}
			status = &parsed
		default:
			return 2, fmt.Errorf("unknown option for reopen: %s", args[i])
		}
	}

	service := app.ReopenItemService{}
	item, err := service.Run(app.ReopenItemInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		Summary:    summary,
		NextAction: nextAction,
		Status:     status,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemReopenedBrief(item))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemReopenedPrompt(item))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runNext(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("next", globals.format)
		}
	}

	agent := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --agent")
			}
			agent = args[i]
		default:
			return 2, fmt.Errorf("unknown option for next: %s", args[i])
		}
	}

	service := app.NextItemService{}
	result, err := service.Run(app.NextItemInput{
		RepoPath: globals.repoPath,
		Agent:    agent,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.NextItemBrief(result))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.NextItemPrompt(result))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runInbox(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("inbox", globals.format)
		}
	}

	agent := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --agent")
			}
			agent = args[i]
		default:
			return 2, fmt.Errorf("unknown option for inbox: %s", args[i])
		}
	}

	service := app.InboxService{}
	results, err := service.Run(app.InboxInput{
		RepoPath: globals.repoPath,
		Agent:    agent,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.InboxBrief(results))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.InboxPrompt(results))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(results)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runLink(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("link", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj link <id> --depends-on <id>")
	}

	itemID := args[0]
	dependencyID := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--depends-on":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --depends-on")
			}
			dependencyID = args[i]
		default:
			return 2, fmt.Errorf("unknown option for link: %s", args[i])
		}
	}
	if dependencyID == "" {
		return 2, errors.New("usage: aj link <id> --depends-on <id>")
	}

	service := app.LinkDependencyService{}
	item, err := service.Run(app.LinkDependencyInput{
		RepoPath:    globals.repoPath,
		ItemID:      itemID,
		DependsOnID: dependencyID,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ItemLinkedBrief(item, dependencyID))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ItemLinkedPrompt(item, dependencyID))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runChanges(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("changes", globals.format)
		}
	}

	itemID := ""
	limit := 20
	var since *time.Time

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--item":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --item")
			}
			itemID = args[i]
		case "--since":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --since")
			}
			parsed, err := time.Parse(time.RFC3339, args[i])
			if err != nil {
				return 2, fmt.Errorf("invalid --since value %q", args[i])
			}
			since = &parsed
		case "--limit":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --limit")
			}
			parsed, err := strconv.Atoi(args[i])
			if err != nil || parsed < 0 {
				return 2, fmt.Errorf("invalid --limit value %q", args[i])
			}
			limit = parsed
		default:
			return 2, fmt.Errorf("unknown option for changes: %s", args[i])
		}
	}

	service := app.ChangesService{}
	events, err := service.Run(app.ChangesInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		Since:    since,
		Limit:    limit,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ChangesBrief(events))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ChangesPrompt(events))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(events)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runReady(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("ready", globals.format)
		}
	}

	agent := ""
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--agent":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --agent")
			}
			agent = args[i]
		default:
			return 2, fmt.Errorf("unknown option for ready: %s", args[i])
		}
	}

	service := app.ReadyService{}
	results, err := service.Run(app.ReadyInput{
		RepoPath: globals.repoPath,
		Agent:    agent,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.ReadyBrief(results))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.ReadyPrompt(results))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(results)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJira(globals globalOptions, args []string) (int, error) {
	if len(args) == 0 {
		return r.renderCommandHelp("jira", globals.format)
	}
	if args[0] == "--help" || args[0] == "-h" {
		return r.renderCommandHelp("jira", globals.format)
	}

	switch args[0] {
	case "pull":
		return r.runJiraPull(globals, args[1:])
	case "push":
		return r.runJiraPush(globals, args[1:])
	case "link":
		return r.runJiraLink(globals, args[1:])
	case "search":
		return r.runJiraSearch(globals, args[1:])
	case "unlink":
		return r.runJiraUnlink(globals, args[1:])
	case "sync":
		return r.runJiraSync(globals, args[1:])
	case "comment":
		return r.runJiraComment(globals, args[1:])
	case "status-map":
		return r.runJiraStatusMap(globals, args[1:])
	case "transitions":
		return r.runJiraTransitions(globals, args[1:])
	default:
		return 2, fmt.Errorf("unknown jira subcommand %q\ntry: aj jira --help", args[0])
	}
}

func (r Runner) runJiraPull(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) != 1 {
		return 2, errors.New("usage: aj jira pull <key>")
	}

	service := app.ImportJiraIssueService{}
	result, err := service.Run(app.ImportJiraIssueInput{
		RepoPath: globals.repoPath,
		IssueKey: args[0],
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if result.AlreadyLinked {
			_, err = fmt.Fprintf(r.stdout, "using existing %s for Jira %s\n", result.Item.ID, args[0])
		} else {
			_, err = fmt.Fprintf(r.stdout, "imported Jira %s as %s\n", args[0], result.Item.ID)
		}
		return 0, err
	case domain.FormatPrompt:
		status := "imported Jira issue"
		if result.AlreadyLinked {
			status = "existing Jira link"
		}
		_, err = fmt.Fprintf(r.stdout, "Status: %s\nJira: %s\nID: %s\nTitle: %s\n", status, args[0], result.Item.ID, result.Item.Title)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraPush(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj jira push <id> [--project <key>] [--type <name>]")
	}

	itemID := args[0]
	projectKey := ""
	issueType := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--project":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --project")
			}
			projectKey = args[i]
		case "--type":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --type")
			}
			issueType = args[i]
		default:
			return 2, fmt.Errorf("unknown option for jira push: %s", args[i])
		}
	}

	service := app.ExportJiraIssueService{}
	result, err := service.Run(app.ExportJiraIssueInput{
		RepoPath:   globals.repoPath,
		ItemID:     itemID,
		ProjectKey: projectKey,
		IssueType:  issueType,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if result.AlreadyLinked {
			_, err = fmt.Fprintf(r.stdout, "already linked %s to Jira %s\n", result.Item.ID, result.Item.Jira.Key)
		} else {
			_, err = fmt.Fprintf(r.stdout, "exported %s to Jira %s\n", result.Item.ID, result.Item.Jira.Key)
		}
		return 0, err
	case domain.FormatPrompt:
		status := "exported to Jira"
		if result.AlreadyLinked {
			status = "already linked to Jira"
		}
		_, err = fmt.Fprintf(r.stdout, "Status: %s\nID: %s\nJira: %s\n", status, result.Item.ID, result.Item.Jira.Key)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraLink(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) < 2 {
		return 2, errors.New("usage: aj jira link <id> <key> [--replace]")
	}

	itemID := args[0]
	issueKey := args[1]
	replace := false
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--replace":
			replace = true
		default:
			return 2, fmt.Errorf("unknown option for jira link: %s", args[i])
		}
	}

	service := app.LinkJiraIssueService{}
	result, err := service.Run(app.LinkJiraIssueInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		IssueKey: issueKey,
		Replace:  replace,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if result.AlreadyLinked {
			_, err = fmt.Fprintf(r.stdout, "already linked %s to Jira %s\n", result.Item.ID, result.Item.Jira.Key)
		} else {
			_, err = fmt.Fprintf(r.stdout, "linked %s to Jira %s\n", result.Item.ID, result.Item.Jira.Key)
		}
		return 0, err
	case domain.FormatPrompt:
		status := "linked to Jira"
		if result.AlreadyLinked {
			status = "already linked to Jira"
		}
		_, err = fmt.Fprintf(r.stdout, "Status: %s\nID: %s\nJira: %s\n", status, result.Item.ID, result.Item.Jira.Key)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraSearch(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj jira search <terms...> [--limit <n>] [--project <key>] | aj jira search --jql <query> [--limit <n>]")
	}

	queryParts := make([]string, 0, len(args))
	rawJQL := ""
	projectKey := ""
	limit := 10
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --limit")
			}
			value, err := strconv.Atoi(args[i])
			if err != nil || value <= 0 {
				return 2, fmt.Errorf("invalid limit %q", args[i])
			}
			limit = value
		case "--project":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --project")
			}
			projectKey = args[i]
		case "--jql":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --jql")
			}
			rawJQL = args[i]
		default:
			queryParts = append(queryParts, args[i])
		}
	}
	if strings.TrimSpace(rawJQL) != "" && len(queryParts) > 0 {
		return 2, errors.New("usage: aj jira search <terms...> [--limit <n>] [--project <key>] | aj jira search --jql <query> [--limit <n>]")
	}
	if strings.TrimSpace(rawJQL) != "" && strings.TrimSpace(projectKey) != "" {
		return 2, errors.New("cannot combine --project with --jql; include the project filter directly in the JQL")
	}
	if strings.TrimSpace(rawJQL) == "" && len(queryParts) == 0 {
		return 2, errors.New("usage: aj jira search <terms...> [--limit <n>] [--project <key>] | aj jira search --jql <query> [--limit <n>]")
	}

	service := app.SearchJiraIssuesService{}
	result, err := service.Run(app.SearchJiraIssuesInput{
		RepoPath:   globals.repoPath,
		Query:      strings.Join(queryParts, " "),
		JQL:        rawJQL,
		ProjectKey: projectKey,
		Limit:      limit,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.JiraSearchBrief(result))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.JiraSearchPrompt(result))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraUnlink(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj jira unlink <id> [--force]")
	}

	itemID := args[0]
	force := false
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--force":
			force = true
		default:
			return 2, fmt.Errorf("unknown option for jira unlink: %s", args[i])
		}
	}

	service := app.UnlinkJiraIssueService{}
	item, err := service.Run(app.UnlinkJiraIssueInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		Force:    force,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintf(r.stdout, "unlinked %s from Jira\n", item.ID)
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintf(r.stdout, "Status: unlinked from Jira\nID: %s\nSummary: %s\n", item.ID, item.Summary)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraSync(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj jira sync <id> [--dry-run] [--resolve keep-local|keep-remote]")
	}

	itemID := args[0]
	dryRun := false
	resolve := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--dry-run":
			dryRun = true
		case "--resolve":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --resolve")
			}
			resolve = args[i]
		default:
			return 2, fmt.Errorf("unknown option for jira sync: %s", args[i])
		}
	}

	service := app.SyncJiraIssueService{}
	result, err := service.Run(app.SyncJiraIssueInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		DryRun:   dryRun,
		Resolve:  resolve,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		if dryRun {
			_, err = fmt.Fprintf(r.stdout, "dry-run jira sync %s direction=%s\n", result.Item.ID, result.Direction)
		} else {
			_, err = fmt.Fprintf(r.stdout, "synced %s with Jira %s direction=%s\n", result.Item.ID, result.Item.Jira.Key, result.Direction)
		}
		return 0, err
	case domain.FormatPrompt:
		status := "synced with Jira"
		if dryRun {
			status = "dry-run Jira sync"
		}
		_, err = fmt.Fprintf(r.stdout, "Status: %s\nID: %s\nJira: %s\nDirection: %s\n", status, result.Item.ID, result.Item.Jira.Key, result.Direction)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraComment(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) == 0 {
		return 2, errors.New("usage: aj jira comment <id> --summary <summary>")
	}

	itemID := args[0]
	summary := ""
	for i := 1; i < len(args); i++ {
		switch args[i] {
		case "--summary":
			i++
			if i >= len(args) {
				return 2, errors.New("missing value for --summary")
			}
			summary = args[i]
		default:
			return 2, fmt.Errorf("unknown option for jira comment: %s", args[i])
		}
	}
	if strings.TrimSpace(summary) == "" {
		return 2, errors.New("usage: aj jira comment <id> --summary <summary>")
	}

	service := app.CommentJiraIssueService{}
	item, err := service.Run(app.CommentJiraIssueInput{
		RepoPath: globals.repoPath,
		ItemID:   itemID,
		Summary:  summary,
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintf(r.stdout, "commented on Jira %s from %s\n", item.Jira.Key, item.ID)
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintf(r.stdout, "Status: commented on Jira\nID: %s\nJira: %s\nSummary: %s\n", item.ID, item.Jira.Key, summary)
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(item)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) postLifecycleJiraComment(repoPath, itemID, summary string) (domain.Item, error) {
	service := app.CommentJiraIssueService{}
	return service.Run(app.CommentJiraIssueInput{
		RepoPath: repoPath,
		ItemID:   itemID,
		Summary:  summary,
	})
}

func (r Runner) lifecycleJiraCommentDefault(repoPath, command string) (bool, error) {
	cfg, err := config.Load(repoPath)
	if err != nil {
		return false, err
	}
	switch command {
	case "done":
		return cfg.Jira.Lifecycle.CommentOnDone, nil
	case "block":
		return cfg.Jira.Lifecycle.CommentOnBlock, nil
	case "handoff":
		return cfg.Jira.Lifecycle.CommentOnHandoff, nil
	default:
		return false, fmt.Errorf("unsupported jira lifecycle policy %q", command)
	}
}

func (r Runner) ensureItemLinkedToJira(repoPath, itemID string) error {
	service := app.ShowItemService{}
	item, err := service.Run(repoPath, itemID)
	if err != nil {
		return err
	}
	if item.Jira == nil || strings.TrimSpace(item.Jira.Key) == "" {
		return fmt.Errorf("item %s is not linked to Jira", item.ID)
	}
	return nil
}

func (r Runner) runJiraStatusMap(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) != 0 {
		return 2, errors.New("usage: aj jira status-map")
	}

	service := app.JiraStatusMapService{}
	result, err := service.Run(globals.repoPath)
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.JiraStatusMapBrief(result))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.JiraStatusMapPrompt(result))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runJiraTransitions(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("jira", globals.format)
		}
	}
	if len(args) != 1 {
		return 2, errors.New("usage: aj jira transitions <id>")
	}

	service := app.JiraTransitionsService{}
	result, err := service.Run(app.JiraTransitionsInput{
		RepoPath: globals.repoPath,
		ItemID:   args[0],
	})
	if err != nil {
		return 1, err
	}

	switch globals.format {
	case domain.FormatBrief:
		_, err = fmt.Fprintln(r.stdout, render.JiraTransitionsBrief(result))
		return 0, err
	case domain.FormatPrompt:
		_, err = fmt.Fprintln(r.stdout, render.JiraTransitionsPrompt(result))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(result)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runHelp(globals globalOptions, args []string) (int, error) {
	if len(args) == 0 {
		return r.renderRootHelp(globals.format)
	}

	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("help", globals.format)
		}
	}

	if args[0] == "search" {
		if len(args) < 2 {
			return 2, errors.New("usage: aj help search <term>")
		}
		return r.renderHelpSearch(globals.format, strings.Join(args[1:], " "))
	}

	if len(args) == 1 {
		if _, ok := r.help.Command(args[0]); ok {
			return r.renderCommandHelp(args[0], globals.format)
		}
		if _, ok := r.help.Workflow(args[0]); ok {
			return r.renderWorkflow(globals.format, args[0])
		}
		if _, ok := r.help.ExampleSet(args[0]); ok {
			return r.renderExampleSet(globals.format, args[0])
		}
		if _, ok := r.help.GlossaryEntry(args[0]); ok {
			return r.renderGlossaryEntry(globals.format, args[0])
		}
	}

	if len(args) == 2 {
		switch args[0] {
		case "workflows":
			return r.renderWorkflow(globals.format, args[1])
		case "examples":
			return r.renderExampleSet(globals.format, args[1])
		case "glossary":
			return r.renderGlossaryEntry(globals.format, args[1])
		}
	}

	return 2, fmt.Errorf("unknown help topic %q", strings.Join(args, " "))
}

func (r Runner) runCommands(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("commands", globals.format)
		}
	}

	if len(args) > 0 {
		return 2, errors.New("usage: aj commands")
	}

	commands := r.help.Commands()
	switch globals.format {
	case domain.FormatBrief, domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.CommandsBrief(commands))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(commands)
	default:
		return 2, fmt.Errorf("unsupported format %q", globals.format)
	}
}

func (r Runner) runWorkflows(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("workflows", globals.format)
		}
	}

	if len(args) == 0 {
		workflows := r.help.Workflows()
		switch globals.format {
		case domain.FormatBrief, domain.FormatPrompt:
			_, err := fmt.Fprintln(r.stdout, render.WorkflowsBrief(workflows))
			return 0, err
		case domain.FormatJSON:
			return r.renderJSON(workflows)
		default:
			return 2, fmt.Errorf("unsupported format %q", globals.format)
		}
	}

	if len(args) > 1 {
		return 2, errors.New("usage: aj workflows [topic]")
	}

	return r.renderWorkflow(globals.format, args[0])
}

func (r Runner) runExamples(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("examples", globals.format)
		}
	}

	if len(args) == 0 {
		exampleSets := r.help.ExampleSets()
		switch globals.format {
		case domain.FormatBrief, domain.FormatPrompt:
			_, err := fmt.Fprintln(r.stdout, render.ExampleSetsBrief(exampleSets))
			return 0, err
		case domain.FormatJSON:
			return r.renderJSON(exampleSets)
		default:
			return 2, fmt.Errorf("unsupported format %q", globals.format)
		}
	}

	if len(args) > 1 {
		return 2, errors.New("usage: aj examples [topic]")
	}

	return r.renderExampleSet(globals.format, args[0])
}

func (r Runner) runGlossary(globals globalOptions, args []string) (int, error) {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return r.renderCommandHelp("glossary", globals.format)
		}
	}

	if len(args) == 0 {
		entries := r.help.Glossary()
		switch globals.format {
		case domain.FormatBrief, domain.FormatPrompt:
			_, err := fmt.Fprintln(r.stdout, render.GlossaryBrief(entries))
			return 0, err
		case domain.FormatJSON:
			return r.renderJSON(entries)
		default:
			return 2, fmt.Errorf("unsupported format %q", globals.format)
		}
	}

	if len(args) > 1 {
		return 2, errors.New("usage: aj glossary [term]")
	}

	return r.renderGlossaryEntry(globals.format, args[0])
}

func (r Runner) renderCommandHelp(name string, format domain.OutputFormat) (int, error) {
	doc, ok := r.help.Command(name)
	if !ok {
		return 2, fmt.Errorf("unknown command %q", name)
	}

	switch format {
	case domain.FormatBrief:
		_, err := fmt.Fprintln(r.stdout, render.CommandHelpBrief(doc))
		return 0, err
	case domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.CommandHelpPrompt(doc))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(doc)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) renderWorkflow(format domain.OutputFormat, topic string) (int, error) {
	workflow, ok := r.help.Workflow(topic)
	if !ok {
		return 2, fmt.Errorf("unknown workflow %q", topic)
	}

	switch format {
	case domain.FormatBrief:
		_, err := fmt.Fprintln(r.stdout, render.WorkflowsBrief([]help.WorkflowDoc{workflow}))
		return 0, err
	case domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.WorkflowPrompt(workflow))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(workflow)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) renderExampleSet(format domain.OutputFormat, topic string) (int, error) {
	exampleSet, ok := r.help.ExampleSet(topic)
	if !ok {
		return 2, fmt.Errorf("unknown example topic %q", topic)
	}

	switch format {
	case domain.FormatBrief, domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.ExampleSetsBrief([]help.ExampleSet{exampleSet}))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(exampleSet)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) renderGlossaryEntry(format domain.OutputFormat, term string) (int, error) {
	entry, ok := r.help.GlossaryEntry(term)
	if !ok {
		return 2, fmt.Errorf("unknown glossary term %q", term)
	}

	switch format {
	case domain.FormatBrief, domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.GlossaryBrief([]help.GlossaryEntry{entry}))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(entry)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) renderHelpSearch(format domain.OutputFormat, term string) (int, error) {
	results := r.help.Search(term)
	switch format {
	case domain.FormatBrief, domain.FormatPrompt:
		_, err := fmt.Fprintln(r.stdout, render.SearchBrief(results))
		return 0, err
	case domain.FormatJSON:
		return r.renderJSON(results)
	default:
		return 2, fmt.Errorf("unsupported format %q", format)
	}
}

func (r Runner) renderJSON(value any) (int, error) {
	payload, err := render.JSON(value)
	if err != nil {
		return 1, err
	}
	_, err = fmt.Fprintln(r.stdout, payload)
	return 0, err
}
