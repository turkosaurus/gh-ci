package log

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/grackleclub/log"
)

// NewFileLogger creates a new file writer, truncating the file if
// it hasn't been written to in overwriteAge.
func NewFileLogger(path string, overwriteAge time.Duration) (*slog.Logger, error) {
	rotated := false
	// ensure dir
	err := os.MkdirAll(filepath.Dir(path), 0o755)
	if err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	// check existing
	info, err := os.Stat(path)
	if err != nil {
		// create if not exists
		if os.IsNotExist(err) {
			_, err := os.Create(path)
			if err != nil {
				return nil, fmt.Errorf("create log file: %w", err)
			}
			info, err = os.Stat(path)
			if err != nil {
				return nil, fmt.Errorf("stat log file after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("stat log file: %w", err)
		}
	}
	// truncate if old
	if time.Since(info.ModTime()) > overwriteAge {
		err = os.Truncate(path, 0)
		if err != nil {
			return nil, fmt.Errorf("rotate log file: %w", err)
		}
		rotated = true
	}
	// open ready file
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	} else if os.Getenv("VERBOSE") != "" {
		level = slog.LevelInfo
	} else {
		level = slog.LevelWarn
	}

	logger, err := log.New(slog.HandlerOptions{Level: level}, f)
	if err != nil {
		return nil, fmt.Errorf("create logger: %w", err)
	}
	logger.Debug("initialized text file logger",
		"path", path,
		"level", level.String(),
	)
	if rotated {
		logger.Debug("prior logs rotated due to age")
	}
	return logger, nil
}
