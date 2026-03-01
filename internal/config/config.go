package config

import (
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config holds the application configuration
type Config struct {
	Repos           []string `yaml:"repos"`
	PrimaryBranch   string   `yaml:"default_branch"`      // e.g. "main"
	RefreshInterval int      `yaml:"refresh_interval"`    // seconds
	MsgTimeout      int      `yaml:"default_msg_timeout"` // seconds
	LogPath         string   `yaml:"log_path"`            // optional log file path
	LogRotateAge    int      `yaml:"log_rotate_age"`      // truncate log flie if not modified (seconds)
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
	}
}

// Load loads configuration from file or auto-detects from git
func Load() (*Config, error) {
	cfg := DefaultConfig()

	// Try to load config file
	configPath := configPath()
	if data, err := os.ReadFile(configPath); err == nil {
		if err := yaml.Unmarshal(data, cfg); err != nil {
			return nil, err
		}
		if cfg.PrimaryBranch == "" {
			cfg.PrimaryBranch = "main"
		}
		if len(cfg.Repos) > 0 {
			return cfg, nil
		}
	}

	// Auto-detect from current git repo
	repo, err := detectGitRepo()
	if err == nil && repo != "" {
		cfg.Repos = []string{repo}
	}

	return cfg, nil
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

// detectGitRepo attempts to detect the GitHub repo from git remote
func detectGitRepo() (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return parseGitRemote(strings.TrimSpace(string(output))), nil
}

// parseGitRemote extracts owner/repo from a git remote URL
func parseGitRemote(url string) string {
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
