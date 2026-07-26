package server

import "log/slog"

func logError(message string, err error, fields ...any) {
	slog.Error(message, append(fields, "error", err)...)
}

func logInfo(message string, fields ...any) {
	slog.Info(message, fields...)
}

func logWarn(message string, fields ...any) {
	slog.Warn(message, fields...)
}
