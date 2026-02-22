# gh-ci

A terminal dashboard for GitHub Actions, built as a [`gh`](https://cli.github.com/) extension.

## Install

```bash
gh extension install turkosaurus/gh-ci
```

Or build from source:

```bash
git clone https://github.com/turkosaurus/gh-ci.git
cd gh-ci
go build -o gh-ci
```

## Usage

Run from inside a GitHub repository:

```bash
gh ci
```

Auto-detects your repo from the current directory. See [Configuration](#configuration) to watch multiple repos.

## Keys

### Main view

| Key | Action |
|-----|--------|
| `↑`/`k`  `↓`/`j` | Navigate |
| `h`/`←`  `l`/`→` | Move between panels |
| `g`/`Home`  `G`/`End` | Top / bottom |
| `PgUp`/`Ctrl+b`  `PgDn`/`Ctrl+f` | Page up / down |
| `Enter` | Select / open branch picker |
| `r` | Re-run workflow |
| `c` | Cancel (in-progress only) |
| `d` | Dispatch workflow |
| `o` | Open in browser |
| `R` | Refresh |
| `q`/`Ctrl+c` | Quit |

### Log viewer

| Key | Action |
|-----|--------|
| `↑`/`k`  `↓`/`j` | Scroll |
| `g`  `G` | Top / bottom |
| `Ctrl+u`  `Ctrl+d` | Half page |
| `/` | Search |
| `n`  `p` | Next / prev match |
| `h`/`Esc`/`⌫` | Back |

## Configuration

`~/.config/gh-ci/config.yml`:

```yaml
repos:
  - owner/repo1
  - owner/repo2
refresh_interval: 30  # seconds (default: 2)
```

## Requirements

- [GitHub CLI](https://cli.github.com/) installed and authenticated
- Go 1.24+ (source builds only)

## License

MIT
