package log

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLog(t *testing.T) {
	tmpDir := t.TempDir()
	logPath := filepath.Join(tmpDir, "test.log")

	os.Setenv("DEBUG", "1")

	logger, err := NewFileLogger(logPath, time.Hour)
	require.NoError(t, err)

	logger.Debug("test debug message", "key", "value")

	content, err := os.ReadFile(logPath)
	require.NoError(t, err)

	assert.Contains(t, string(content), "DBG")
	t.Logf("------- log content -------\n%s", string(content))
}
