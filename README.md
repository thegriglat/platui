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
go install github.com/thegriglat/platui@latest
```

Or build from source:

```sh
git clone https://github.com/thegriglat/platui.git
cd platui
go build -o platui .
```

## Configuration

platui reads `$XDG_CONFIG_HOME/platui/config.yml` (usually `~/.config/platui/config.yml`);
use `--config <path>` to point elsewhere. A minimal config:

```yaml
base_url: https://api.plane.so   # your instance URL for self-hosted Plane
api_key: plane_api_xxx           # or: api_key_env: PLANE_API_KEY
workspace: my-team               # workspace slug from the Plane URL
default_project: PROJ            # optional: project identifier to open on start
branch:
  template: "{identifier}-{sequence}-{slug}"
```

Create an API key in Plane under *Profile settings → Personal Access Tokens*.
All options (tabs, saved filters, worktrees, `workflow`, `gitlab`, theme, refresh interval, cache) are
documented with defaults in [`config.example.yml`](config.example.yml). Unknown keys are an error. Only one
workspace per config is supported.

Notes:

- Set either `api_key` or `api_key_env`, not both; the key never appears in error messages.
  If the key is stored in the file, keep it private: `chmod 600 ~/.config/platui/config.yml`.
- `tabs` — one tab per project state; by default the `cancelled` group is hidden (`hide_groups`) and the first tab is "All".
- `filters` — up to 9 saved filters, switched with `F` or the command palette (digits are not used: they belong to tabs).
- `gitlab` — merge requests go through `glab`; `host` and `repo` default to the `remote` URL (`branch.remote`, `origin`).
- `refresh.interval` is `0` (off) or at least `30s` (Plane API limit is 60 requests/min); default `2m`.
- `commands` — your own shell commands bound to keys (`name`, `key`, `command`, `confirm`, `output`: `status`|`window`|`none`).
  Placeholders `{id} {key} {branch} {url} {title} {mr_url}` are substituted already shell-quoted: do not put them in quotes
  (a placeholder inside quotes is a config error). A key that collides with a built-in key is an error.
- `ui.mouse` (default `true`) and `ui.preview_width` (percent, 20..80, default 45; `<` `>` `z` change it at run time).
- `todo.path` — personal todo file; default `$XDG_DATA_HOME/platui/todo.md` or `~/.local/share/platui/todo.md`.
- `NO_COLOR` (or `theme.no_color: true`) disables colors.

### Custom CA (self-hosted Plane over HTTPS)

If Plane uses a certificate from your own CA, add it to the system ones with `http.ca_file: ~/certs/my-ca.crt` (PEM, one or more certificates). `glab` and `git push` use their own CA settings: `glab config set ca_cert <file> --host <host>`, `git config http.sslCAInfo <file>`.

## Key bindings

Press `?` or `h` in the app for the full list.

| Group | Keys |
|---|---|
| Navigation | `↑/↓`, `PgUp/PgDn`, `Home/End`, `Enter` open, `Esc` back, `Tab`/`Shift+Tab` switch pane, `Ctrl+←/→` or `Alt+1…9` switch tab, `P` projects |
| Search | `/` search, `Ctrl+G` go to ID, `Ctrl+P` command palette |
| Layout | `<` / `>` resize preview, `z` zoom preview, `→` / `←` expand / collapse subtasks in the table |
| Work item | `s` state, `p` priority, `a` assign to me, `e` edit description, `c` comment, `n` new, `y` copy, `O` open in browser, `t` timer, `l` links from description and comments, `B` bookmark, `Ctrl+O` back to the previously opened item, `space` select |
| Git | `b` create branch, `S` start work, `W` ready for review, `M` go to MR, `X` clean up branches |
| Views | `v` kanban, `m` mine, `F` saved filters, `T` personal todo, `o` sort, `R` refresh from API, `q` quit |

## Usage

Run `platui` to open the TUI: tabs by state on top, a table of work items below, a preview pane for the
selected item. Common flags: `--config <path>`, `--repo <dir>`, `--project <ID>`, `--item PROJ-42`
(open on an item).

Git scenarios use the repository in the current directory (or `--repo`): `b` creates a branch from
`branch.template`, `S` creates it, moves the item to the `workflow.start_state` (default: first *started* state)
and assigns it to you, `W` pushes the branch and opens a merge request with `glab`, `M` finds the item's merge
request and opens it in the browser (`Go to MR`).

### CLI subcommands

`ls` (`--mine`, `--state`, `--all`, `--json`), `show <ref>` (`--comments`, `--json`), `search <query…>`
(`--server`, `--json`), `branch <ref>` (`--print`, `--worktree`), `start <ref>` (`--worktree`), `review [ref]`,
`mr [ref]` (`--open`, `--json`), `prompt` (`--format`), `todo [ls|add <text…>|done <N>|undone <N>]`
(personal todo, local file, no network), `hook install`, `version`.

### Confirmation of destructive actions

In the TUI such actions (bulk changes, state moves, deleting branches or todo items, running custom commands, overwriting
a description changed in Plane) ask for confirmation in a `y/N` dialog: only `y`/`Y` confirms, anything else cancels.
The CLI has a global `--yes` flag: with it destructive actions run without asking; without it the CLI asks `[y/N]`
and, when stdin is not a terminal, refuses with "confirmation required: --yes".

### Todo view

`T` opens a personal todo list stored in a local Markdown file (`todo.path`). Keys: `↑↓`/`PgUp`/`PgDn`/`Home`/`End` move,
`n` new, `space` done / not done, `e` edit, `d` delete (with confirmation), `Shift+↑/↓` reorder, `C` turn the entry into a
Plane work item, `Enter` go to the linked item, `Esc` back.

### Cache

Data is cached on disk (`cache.dir`, default `$XDG_CACHE_HOME/platui` or `~/.cache/platui`; disable with
`cache.enabled: false`) so the next start shows the last data immediately and refreshes in the background.
Cache files are written with mode `0600`.

## Limitations

- One workspace per config (several workspaces/profiles are not supported).
- Merge requests: GitLab only, via `glab`; `glab` must be installed and authenticated.
- Vim-style key bindings are not provided.

## Development

```sh
go build ./...
go vet ./...
go test ./...
```

Tests do not touch the network: the Plane API is faked with `httptest`, git runs against temporary repositories.

## License

[MIT](LICENSE)
