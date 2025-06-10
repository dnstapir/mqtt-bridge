package shared

type ILogger interface {
	Debug(fmtStr string, vals ...any)
	Info(fmtStr string, vals ...any)
	Warning(fmtStr string, vals ...any)
	Error(fmtStr string, vals ...any)
}
