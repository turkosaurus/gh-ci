package main

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
)

func newFileLogger() (*slog.Logger, error) {
	logFile := os.Getenv("GH_CI_LOG_FILE")
	if logFile == "" {
		home, err := os.UserHomeDir()
		logFile = filepath.Join(home, ".config", "gh-ci", "ci.log") // TODO: integrate with config
		if err != nil {
			return nil, fmt.Errorf("determine home directory: %w", err)
		}
		err = os.MkdirAll(filepath.Dir(logFile), 0755)
		if err != nil {
			return nil, fmt.Errorf("create log directory: %w", err)
		}
	}

	f, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("open log file: %w", err)
	}

	level := slog.LevelInfo
	if os.Getenv("DEBUG") != "" {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(f, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	logger.Debug("initialized text file logger",
		"path", logFile,
		"level", level.String(),
	)

	// TODO: truncate old logs
	// maxAge := 1 * time.Minute
	// // scan through and delete old log entries
	// scanner := bufio.NewScanner(f)
	// for scanner.Scan() {
	// 	line := scanner.Text()
	// 	fields := strings.Fields(line)
	// 	if len(fields) == 0 {
	// 		continue
	// 	}
	// 	timestampStr := fields[0]
	// 	t, err := time.Parse(time.RFC3339, timestampStr)
	// 	if err != nil {
	// 		slog.Error("log rotate: parse log timestamp",
	// 			"file", logFile,
	// 			"line", line,
	// 			"error", err,
	// 		)
	// 		continue
	// 	}
	// 	if time.Since(t) > maxAge {
	// 		slog.Info("log rotate: old log entry",
	// 			"file", logFile,
	// 			"line", line,
	// 			"age", time.Since(t),
	// 		)
	// 	}
	// }
	return logger, nil
}
