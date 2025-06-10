package logging

import (
	"fmt"
	"log/slog"
	"os"
)

type logger struct {
	logger *slog.Logger
}

func Create(debug, quiet bool) logger {
	var programLevel = new(slog.LevelVar) // Info by default

	/* quiet overrides debug */
	if debug {
		programLevel.Set(slog.LevelDebug)
	}
	if quiet {
		programLevel.Set(slog.LevelWarn)
	}

	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})

	l := slog.New(h)

	return logger{logger: l}
}

func (l logger) Debug(fmtStr string, vals ...any) {
	l.logger.Debug(format(fmtStr, vals))
}

func (l logger) Info(fmtStr string, vals ...any) {
	l.logger.Info(format(fmtStr, vals))
}

func (l logger) Warning(fmtStr string, vals ...any) {
	l.logger.Warn(format(fmtStr, vals))
}

func (l logger) Error(fmtStr string, vals ...any) {
	l.logger.Error(format(fmtStr, vals))
}

func format(fmtStr string, a []any) string {
	if len(a) == 0 {
		return fmtStr
	}

	return fmt.Sprintf(fmtStr, a...)
}
