package fake

import "fmt"

type logger struct {
}

func Logger() *logger {
    logger := new(logger)
    return logger
}

func (l *logger) Debug(fmtStr string, vals ...any) {
}

func (l *logger) Info(fmtStr string, vals ...any) {
}

func (l *logger) Warning(fmtStr string, vals ...any) {
}

func (l *logger) Error(fmtStr string, vals ...any) {
    panic(format(fmtStr, vals))
}

func format(fmtStr string, a []any) string {
	if len(a) == 0 {
		return fmtStr
	}

	return fmt.Sprintf(fmtStr, a...)
}
