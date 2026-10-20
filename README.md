# platui

A terminal UI for the [Plane](https://plane.so) project management tool, written in Go.
Browse and search work items, update them without leaving the terminal, and create a git branch
for an issue with a single key.

> **Status: work in progress.** The core packages (Plane API client, config, git, search, cache)
> are being built; the TUI itself is under active development. Expect breaking changes until v1.0.

## Features

- **Plane Cloud and self-hosted** — public REST API, authentication with an API key.
- **Tabs by state** — one tab per project state (Backlog, To Do, In Progress, …) with item counts,
  a table of work items under each tab and a preview pane for the selected item.
- **Instant fuzzy search** by title as you type (fzf algorithm), across all states; jump to an item by `PROJ-42`.
- **Git integration**
  - create a branch for an item from a template (`{identifier}-{sequence}-{slug}`), optionally in a `git worktree`;
  - "start work": branch + move to *In Progress* + assign to yourself;
  - "ready for review": push + open a GitLab merge request via [`glab`](https://gitlab.com/gitlab-org/cli), link it in a comment, move to *In Review*;
  - "go to MR": find the merge request of an item and open it in the browser;
  - detect the item from the current branch name; commit message template / `prepare-commit-msg` hook;
  - clean up branches of closed items.
- **Editing** — change state and priority, assign, create items (one-line quick add: `fix login !high @me #bug`),
  edit descriptions and write comments in `$EDITOR` (Markdown).
- **Developer conveniences** — command palette (`Ctrl+P`), saved filters, kanban view, bulk actions,
  copy ID / link / branch name to the clipboard via OSC 52 (works over SSH), time tracking, background refresh,
  on-disk cache for instant start-up.
- **Scriptable CLI** — `platui ls --mine --json`, `platui show PROJ-42`, `platui branch PROJ-42`,
  `platui start PROJ-42`, `platui mr PROJ-42 --open`, `platui prompt` (for shell prompts / tmux status lines).

## Installation

Requires Go 1.24+. Merge request features need [`glab`](https://gitlab.com/gitlab-org/cli) (GitLab only).

```sh
go install github.com/thegriglat/platui/cmd/platui@latest
```

Or build from source:

```sh
git clone https://github.com/thegriglat/platui.git
cd platui
go build -o platui ./cmd/platui
```

## Configuration

platui reads `$XDG_CONFIG_HOME/platui/config.yml` (usually `~/.config/platui/config.yml`);
use `--config <path>` to point elsewhere. A minimal config:

```yaml
base_url: https://api.plane.so   # your instance URL for self-hosted Plane
api_key_env: PLANE_API_KEY       # or: api_key: plane_api_xxx
workspace: my-team               # workspace slug from the Plane URL
default_project: PROJ            # optional: project identifier to open on start
branch:
  template: "{identifier}-{sequence}-{slug}"
```

Create an API key in Plane under *Profile settings → Personal Access Tokens*.
All options (tabs, saved filters, worktrees, GitLab, theme, refresh interval, cache) will be documented in
`config.example.yml`.

## Key bindings

Press `?` or `h` in the app for the full list.

| Group | Keys |
|---|---|
| Navigation | `↑/↓`, `PgUp/PgDn`, `Home/End`, `Enter` open, `Esc` back, `Tab`/`Shift+Tab` switch pane, `Ctrl+←/→` or `Alt+1…9` switch tab, `P` projects |
| Search | `/` search, `Ctrl+G` go to ID, `Ctrl+P` command palette |
| Work item | `s` state, `p` priority, `a` assign to me, `e` edit description, `c` comment, `n` new, `y` copy, `O` open in browser, `t` timer, `space` select |
| Git | `b` create branch, `S` start work, `W` ready for review, `M` go to MR, `X` clean up branches |
| Views | `v` kanban, `m` mine, `F` saved filters, `o` sort, `R` refresh from API, `q` quit |

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

Tests do not touch the network: the Plane API is faked with `httptest`, git runs against temporary repositories.

## License

[MIT](LICENSE)
