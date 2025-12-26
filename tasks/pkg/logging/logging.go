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

func SetupLogger(env string) *slog.Logger {
	var handler slog.Handler
	switch env {
	case "dev":
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	case "prod":
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	default:
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	}

	return slog.New(handler)
}
