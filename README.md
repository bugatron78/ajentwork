# ajentwork

`ajentwork` is a local-first work tracker for AI agents, with the `aj` CLI as its primary interface.

## What It Does

- Tracks compact work items in a git-friendly `.aj/` directory
- Records append-only events for item history
- Supports leases, dependencies, queueing, history, and ready-work views
- Can be used to dogfood the development of the tool itself

## Build

```bash
go build ./cmd/aj
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
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --version v0.1.2
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

- `aj jira pull <key>`
- `aj jira push <id> [--project <key>] [--type <name>]`
- `aj jira link <id> <key> [--replace]`
- `aj jira unlink <id> [--force]`
- `aj jira status-map`
- `aj jira transitions <id>`
- `aj jira sync <id> [--dry-run] [--resolve keep-local|keep-remote]`
- `aj jira comment <id> --summary "..."`
- `aj take jira <key> --agent <name>`

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
./scripts/build-release.sh v0.1.2
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
4. Run `aj --help` or `man ./share/man/man1/aj.1`.

## Releases

Tagged pushes like `v0.1.0` trigger the GitHub Actions release workflow, which builds artifacts and attaches them to a GitHub Release.

## License

Apache License 2.0. See `LICENSE`.
