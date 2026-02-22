package gh

import "testing"

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    int64
		want string
	}{
		{0, "0s"},
		{45, "45s"},
		{59, "59s"},
		{60, "1m 0s"},
		{90, "1m 30s"},
		{3600, "1h 0m"},
		{3661, "1h 1m"},
	}
	for _, tt := range tests {
		got := FormatDuration(tt.d)
		if got != tt.want {
			t.Errorf("FormatDuration(%d) = %q, want %q", tt.d, got, tt.want)
		}
	}
}

func TestTruncateString(t *testing.T) {
	tests := []struct {
		s      string
		maxLen int
		want   string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"hi", 2, "hi"},
		{"hello", 3, "hel"},
		{"abcdef", 4, "a..."},
	}
	for _, tt := range tests {
		got := TruncateString(tt.s, tt.maxLen)
		if got != tt.want {
			t.Errorf("TruncateString(%q, %d) = %q, want %q", tt.s, tt.maxLen, got, tt.want)
		}
	}
}

func TestSplitRepo(t *testing.T) {
	tests := []struct {
		repo      string
		wantOwner string
		wantName  string
	}{
		{"owner/repo", "owner", "repo"},
		{"just-repo", "", "just-repo"},
		{"", "", ""},
		{"org/my-repo", "org", "my-repo"},
	}
	for _, tt := range tests {
		owner, name := SplitRepo(tt.repo)
		if owner != tt.wantOwner || name != tt.wantName {
			t.Errorf("SplitRepo(%q) = (%q, %q), want (%q, %q)",
				tt.repo, owner, name, tt.wantOwner, tt.wantName)
		}
	}
}
