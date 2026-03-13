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

## Quick Install

Install the latest release binary:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash
```

The installer downloads the matching release artifact and verifies its SHA-256 checksum before installing.

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --version v0.1.1
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/bugatron78/ajentwork/main/scripts/install.sh | bash -s -- --install-dir "$HOME/.local/bin"
```

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

## Jira Adapter

The first Jira adapter slice supports:

- `aj jira pull <key>`
- `aj jira push <id> [--project <key>] [--type <name>]`
- `aj jira link <id> <key>`
- `aj jira sync <id> [--dry-run] [--resolve keep-local|keep-remote]`
- `aj jira comment <id> --summary "..."`
- `aj take jira <key> --agent <name>`

`aj jira sync` will also try to move the remote Jira issue to the mapped Jira status when the local `aj` item status has changed and Jira exposes a matching transition.

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
./scripts/build-release.sh v0.1.1
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
3. Run `aj --help`.

## Releases

Tagged pushes like `v0.1.0` trigger the GitHub Actions release workflow, which builds artifacts and attaches them to a GitHub Release.

## License

Apache License 2.0. See `LICENSE`.
