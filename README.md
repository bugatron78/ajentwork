# ajentwork

`ajentwork` is a local-first work tracker for AI agents, with the `aj` CLI as its primary interface.

## What It Does

- Tracks compact work items in a git-friendly `.aj/` directory
- Records append-only events for item history
- Supports leases, dependencies, queueing, history, and ready-work views
- Can be used to dogfood the development of the tool itself

## Writing Good Agent Tickets

`aj` works best when tickets and updates are written as handoff-quality context for the next agent, not just as terse status markers.

- Titles should name the concrete problem or outcome.
- Goals should explain why the work matters, important constraints, and where acceptance evidence will come from.
- Use `aj new --accept ... --constraint ... --risk ... --file ... --verify ...` when those details would help another agent execute accurately without rediscovering the same context.
- Progress, block, handoff, and done summaries should say what changed, what was learned, and what risk or verification remains.
- Next actions should be specific enough that another agent can start work without rereading the whole repo.

Use `aj workflows authoring` and `aj examples authoring` for copyable examples.

## Evidence And Receipts

Use artifacts when another agent will need proof of what was tried or what happened:

- `aj attach <id> --path <file> --summary "..."` copies a supporting file like a log, patch, or note into `.aj/artifacts/`.
- `aj receipt <id> --summary "..." --command "..." --exit-code <n> [--output <file>]` records a compact execution receipt for a build, test, or verification step.
- `aj artifacts <id>` lists the attached evidence for one item, and `aj show <id>` includes the latest artifact summaries.

## Checkpoints And Handoffs

Use checkpoints when another agent may need to resume work later, even if you are not transferring the lease yet:

- `aj checkpoint <id> --summary "..." [--next "..."] [--risk "..."] [--verify "..."]` records a compact resume point.
- `aj show <id>` surfaces the latest checkpoint summary, remaining risks, and verification guidance.
- `aj handoff ...` still transfers ownership; checkpoints make the handoff payload better before that transfer happens.

## Build

```bash
go build ./cmd/aj
```

Check the installed binary version:

```bash
aj --version
```

Generate the man page:

```bash
go run ./cmd/ajgenman --output docs/aj.1
```

## Quick Install

Install the latest release binary:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash
```

The installer downloads the matching release artifact and verifies its SHA-256 checksum before installing.

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --version v0.1.4
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --install-dir "$HOME/.local/bin"
```

Install without the man page:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --no-man
```

By default, the installer also places `aj.1` in `$HOME/.local/share/man/man1`.

## Homebrew

One-shot install without pre-tapping:

```bash
brew install bugatron78/ajentwork/aj
```

If you want plain `brew install aj`, tap the repo first:

```bash
brew tap bugatron78/ajentwork
brew install aj
```

The published tap lives at `bugatron78/homebrew-ajentwork`, and the formula builds `aj` from source with Go.
It also installs the `aj(1)` man page, so `man aj` works after a Homebrew install.

## Jira Adapter

The first Jira adapter slice supports:

- `aj jira search <terms...> [--limit <n>] [--project <key>]`
- `aj jira pull <key>`
- `aj jira push <id> [--project <key>] [--type <name>]`
- `aj jira link <id> <key> [--replace]`
- `aj jira unlink <id> [--force]`
- `aj jira status-map`
- `aj jira transitions <id>`
- `aj jira sync <id> [--dry-run] [--resolve keep-local|keep-remote]`
- `aj jira comment <id> --summary "..."`
- `aj take jira <key> --agent <name>`

Use `aj jira search ...` before creating or linking new work so agents can check whether a matching Jira issue already exists.
`aj jira sync` will also try to move the remote Jira issue to the mapped Jira status when the local `aj` item status has changed and Jira exposes a matching transition.
Use `aj jira status-map` and `aj jira transitions <id>` to inspect the configured mapping and the live remote workflow before syncing.
Use `aj jira unlink <id>` before moving an item to a different Jira issue, or `aj jira link <id> <key> --replace` when that relink is intentional.
Linked lifecycle commands can also emit milestone comments directly with `--jira-comment` on `aj done`, `aj block`, and `aj handoff`.
Repos can also default those comments with:

```toml
[jira.lifecycle]
comment_on_done = true
comment_on_block = true
comment_on_handoff = true
```

Use `--no-jira-comment` on a specific command to suppress the repo default.

Set credentials through environment variables:

```bash
export AJ_JIRA_EMAIL="you@example.com"
export AJ_JIRA_API_TOKEN="..."
```

Then enable Jira in `.aj/config.toml` and set at least:

```toml
[jira]
enabled = true
base_url = "https://your-domain.atlassian.net"
project = "ABC"
```

## Release Artifacts

Build shareable binaries for macOS and Linux:

```bash
./scripts/build-release.sh
```

Or build for a specific version label:

```bash
./scripts/build-release.sh v0.1.4
```

Artifacts are written to `dist/`:

- `aj_<version>_darwin_amd64.tar.gz`
- `aj_<version>_darwin_arm64.tar.gz`
- `aj_<version>_linux_amd64.tar.gz`
- `aj_<version>_linux_arm64.tar.gz`
- `aj_<version>_checksums.txt`

## Install From An Artifact

1. Extract the archive for your platform.
2. Move `aj` onto your `PATH`.
3. Optionally copy `share/man/man1/aj.1` into your preferred man directory.
4. Run `aj --version`, `aj --help`, or `man ./share/man/man1/aj.1`.

## Releases

Tagged pushes like `v0.1.0` trigger the GitHub Actions release workflow, which builds artifacts and attaches them to a GitHub Release.

## License

Apache License 2.0. See `LICENSE`.
