package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/grackleclub/log"
)

// newFileLogger sets up a "text" or "json" logger.
func newFileLogger(format string) (*slog.Logger, error) {
	rotated := false
	maxAge := 1 * time.Second // FIXME: be realistic

	// set path
	logFile := os.Getenv("GH_CI_LOG_FILE")
	if logFile == "" {
		home, err := os.UserHomeDir()
		logFile = filepath.Join(home, ".config", "gh-ci", "ci.log") // TODO: integrate with config
		if err != nil {
			return nil, fmt.Errorf("determine log dir: %w", err)
		}
	}
	// ensure dir
	err := os.MkdirAll(filepath.Dir(logFile), 0o755)
	if err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	// check existing
	info, err := os.Stat(logFile)
	if err != nil {
		// create if not exists
		if os.IsNotExist(err) {
			_, err := os.Create(logFile)
			if err != nil {
				return nil, fmt.Errorf("create log file: %w", err)
			}
			info, err = os.Stat(logFile)
			if err != nil {
				return nil, fmt.Errorf("stat log file after creation: %w", err)
			}
		} else {
			return nil, fmt.Errorf("stat log file: %w", err)
		}
	}
	// truncate if old
	if time.Since(info.ModTime()) > maxAge {
		err = os.Truncate(logFile, 0)
		if err != nil {
			return nil, fmt.Errorf("rotate log file: %w", err)
		}
		rotated = true
	}
	// open ready file
	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
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
		"path", logFile,
		"level", level.String(),
	)
	if rotated {
		logger.Debug("prior logs rotated due to age")
	}
	return logger, nil
}
