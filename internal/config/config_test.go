package config

import "testing"

func TestParseGitRemote(t *testing.T) {
	tests := []struct {
		url  string
		want string
	}{
		{"git@github.com:owner/repo.git", "owner/repo"},
		{"https://github.com/owner/repo.git", "owner/repo"},
		{"https://github.com/owner/repo", "owner/repo"},
		{"https://github.com/owner/repo-name.git", "owner/repo-name"},
		{"", ""},
		{"not-a-github-url", ""},
	}
	for _, tt := range tests {
		got := parseGitRemote(tt.url)
		if got != tt.want {
			t.Errorf("parseGitRemote(%q) = %q, want %q", tt.url, got, tt.want)
		}
	}
}
