package config

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const DefaultLogContext = 3

// Config holds the application configuration
type Config struct {
	Repos           []string `yaml:"-"`
	PrimaryBranch   string   `yaml:"default_branch"`      // e.g. "main"
	RefreshInterval int      `yaml:"refresh_interval"`    // seconds
	MsgTimeout      int      `yaml:"default_msg_timeout"` // seconds
	LogPath         string   `yaml:"log_path"`            // optional log file path
	LogRotateAge    int      `yaml:"log_rotate_age"`      // truncate log flie if not modified (seconds)
	LogContext      int      `yaml:"log_context"`         // lines of context around each log search match
	LogWrap         bool     `yaml:"log_wrap"`            // wrap long lines in log viewer
	Theme           string   `yaml:"theme"`               // theme name (e.g. "dracula")
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		Repos:           []string{},
		PrimaryBranch:   "main",
		RefreshInterval: 2,
		MsgTimeout:      3,
		LogPath:         logPath(),
		LogRotateAge:    10, // TODO: change to hours so as to be session useful
		LogContext:      DefaultLogContext,
		LogWrap:         false,
		Theme:           "auto",
	}
}

// Load loads configuration from file and auto-detects repo from git
func Load() (*Config, error) {
	cfg := DefaultConfig()

	configPath := configPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
	}

	repo, err := gitRepoURL()
	if err == nil && repo != "" {
		cfg.Repos = []string{repo}
	}

	return cfg, nil
}

// WriteDefault writes the default config to configPath if no config file exists.
func WriteDefault() error {
	p := configPath()
	if _, err := os.Stat(p); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return err
	}
	data, err := yaml.Marshal(DefaultConfig())
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0o644)
}

// logPath returns the default log path, trying to use a user-specific location if possible
func logPath() string {
	logPath := path.Join("/tmp", "gh-ci", "ci.log")
	home, err := os.UserHomeDir()
	if err == nil {
		logPath = filepath.Join(home, ".config", "gh-ci", "ci.log")
	}
	return logPath
}

// configPath returns the path to the config file
func configPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "gh-ci", "config.yml")
}

// gitRepoURL attempts to detect the GitHub repo from git remote
func gitRepoURL() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return GitOwnerRepo(strings.TrimSpace(string(output))), nil
}

// RepoInfo holds a discovered repo's owner/repo name and its checked-out branch.
type RepoInfo struct {
	Name   string // owner/repo
	Branch string // active branch
}

// DiscoverRepos finds GitHub repos in parent directories up to $HOME.
// The current repo (if any) is always the first element.
func DiscoverRepos() []RepoInfo {
	seen := map[string]bool{}
	var repos []RepoInfo

	// Current repo first
	if current, err := gitRepoURL(); err == nil && current != "" {
		repos = append(repos, RepoInfo{Name: current, Branch: gitBranch(".")})
		seen[current] = true
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return repos
	}
	cwd, err := os.Getwd()
	if err != nil {
		return repos
	}

	dir := filepath.Dir(cwd)
	for dir != home && strings.HasPrefix(dir, home) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			break
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			candidate := filepath.Join(dir, e.Name())
			if _, err := os.Stat(filepath.Join(candidate, ".git")); err != nil {
				continue
			}
			cmd := exec.Command("git", "-C", candidate, "remote", "get-url", "origin")
			out, err := cmd.Output()
			if err != nil {
				continue
			}
			repo := GitOwnerRepo(strings.TrimSpace(string(out)))
			if repo == "" || seen[repo] {
				continue
			}
			seen[repo] = true
			repos = append(repos, RepoInfo{Name: repo, Branch: gitBranch(candidate)})
		}
		dir = filepath.Dir(dir)
	}
	return repos
}

func gitBranch(dir string) string {
	cmd := exec.Command("git", "-C", dir, "branch", "--show-current")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// GitOwnerRepo extracts owner/repo from a git remote URL.
func GitOwnerRepo(url string) string {
	// Handle SSH URLs: git@github.com:owner/repo.git
	if strings.HasPrefix(url, "git@github.com:") {
		url = strings.TrimPrefix(url, "git@github.com:")
		url = strings.TrimSuffix(url, ".git")
		return url
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	if strings.Contains(url, "github.com/") {
		parts := strings.Split(url, "github.com/")
		if len(parts) == 2 {
			repo := strings.TrimSuffix(parts[1], ".git")
			return repo
		}
	}

	return ""
}
