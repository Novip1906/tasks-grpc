package logging

import (
	"log/slog"
	"os"
)

func Err(err error) slog.Attr {
	return slog.Attr{
		Key:   "error",
		Value: slog.StringValue(err.Error()),
	}
}

func DbErr(method string, err error) slog.Attr {
	return slog.Attr{
		Key: "db",
		Value: slog.GroupValue(
			slog.String("method", method),
			slog.String("error", err.Error()),
		),
	}
}

func SetupLogger(level slog.Level) *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}
