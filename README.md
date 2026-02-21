# gh-ci

A terminal UI dashboard for viewing and managing GitHub Actions workflow runs, built as a `gh` CLI extension.

## Features

- **Workflow Runs List**: View recent workflow runs across repositories
- **Color-coded Status**: Visual indicators for success (✓), failure (✗), in-progress (●), and pending (○)
- **Actions**: Re-run, cancel, and open workflows in browser
- **Log Viewer**: View job logs with vim-like navigation
- **Filtering**: Filter by status (all, failed, running, success) and search by name/branch
- **Auto-refresh**: Automatically refreshes workflow data (configurable interval)

## Installation

### As a gh extension

```bash
gh extension install turkosaurus/gh-ci
```

### Build from source

```bash
git clone https://github.com/turkosaurus/gh-ci.git
cd gh-ci
go build -o gh-ci
```

## Usage

Run in a directory with a GitHub repository:

```bash
gh ci
# or if built from source
./gh-ci
```

## Keyboard Shortcuts

### Navigation
| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `g` / `Home` | Go to top |
| `G` / `End` | Go to bottom |
| `PgUp` / `Ctrl+b` | Page up |
| `PgDn` / `Ctrl+f` | Page down |

### Actions
| Key | Action |
|-----|--------|
| `Enter` | View run details |
| `l` | View logs |
| `o` | Open in browser |
| `r` | Re-run workflow |
| `c` | Cancel running workflow |
| `R` | Refresh |

### Filtering
| Key | Action |
|-----|--------|
| `/` | Open search filter |
| `Tab` | Cycle status filter |
| `Esc` | Clear filter / go back |

### General
| Key | Action |
|-----|--------|
| `?` | Toggle help |
| `q` / `Ctrl+c` | Quit |

## Configuration

The extension auto-detects the repository from your current git directory. For watching multiple repositories, create a config file at `~/.config/gh-ci/config.yml`:

```yaml
repos:
  - owner/repo1
  - owner/repo2
refresh_interval: 30  # seconds
```

## Requirements

- Go 1.21+ (for building from source)
- GitHub CLI (`gh`) installed and authenticated
- Git repository with GitHub remote (for auto-detection)

## License

MIT
