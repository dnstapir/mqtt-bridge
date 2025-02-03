package log

import (
    "fmt"
    "os"
    "log/slog"
)

func Initialize(debug bool) {
    var programLevel = new(slog.LevelVar) // Info by default

    if debug {
        programLevel.Set(slog.LevelDebug)
    }

    h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: programLevel})
    slog.SetDefault(slog.New(h))
}

func Debug(fmtStr string, vals ...any) {
    slog.Debug(format(fmtStr, vals))
}

func Info(fmtStr string, vals ...any) {
    slog.Info(format(fmtStr, vals))
}

func Warning(fmtStr string, vals ...any) {
    slog.Warn(format(fmtStr, vals))
}

func Error(fmtStr string, vals ...any) {
    slog.Error(format(fmtStr, vals))
}

func format(fmtStr string, a []any) string {
    // TODO fix, does not look nice with multiple varargs
    if len(a) == 0 {
        return fmtStr
    }

    return fmt.Sprintf(fmtStr, a)
}
