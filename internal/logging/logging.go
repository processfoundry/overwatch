package logging

import (
	"log/slog"
	"os"
)

func Init(level slog.Level) {
	h := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(h))
}
