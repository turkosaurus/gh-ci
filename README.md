# gh-ci
[![Go Test](https://github.com/turkosaurus/gh-ci/actions/workflows/test.yaml/badge.svg)](https://github.com/turkosaurus/gh-ci/actions/workflows/test.yaml) [![Release](https://github.com/turkosaurus/gh-ci/actions/workflows/release.yaml/badge.svg)](https://github.com/turkosaurus/gh-ci/actions/workflows/release.yaml)

terminal dashboard for CI actions/workflows

## installation & usage

### extension (recommended)
Requires [`gh`](https://cli.github.com/) CLI.
```bash
gh extension install turkosaurus/gh-ci
gh ci
```

### source 
Requires [go](https://go.dev).
```bash
git clone https://github.com/turkosaurus/gh-ci.git
cd gh-ci
go build -o gh-ci
./gh-ci
```

## features
- auto-detects current repo and branch
- select any branch
- workflows may be dispatched, rerun, or rerun with debug logs
- logs searchable

## keys

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

### log viewer

| Key | Action |
|-----|--------|
| `↑`/`k`  `↓`/`j` | Scroll |
| `g`  `G` | Top / bottom |
| `Ctrl+u`  `Ctrl+d` | Half page |
| `/` | Search |
| `n`  `p` | Next / prev match |
| `h`/`Esc`/`⌫` | Back |

## config

`~/.config/gh-ci/config.yml`:

```yaml
repos:
  - owner/repo1
  - owner/repo2
refresh_interval: 30  # seconds (default: 2)
```
