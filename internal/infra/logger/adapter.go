package logger

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

// Logger interface definition
type Logger interface {
	Debug(context.Context, ...any)
	Info(context.Context, ...any)
	Warn(context.Context, ...any)
	Error(context.Context, ...any)
}

// ZapLogger construct logger adapter
type ZapLogger struct {
	Logger *zap.Logger
}

// Debug build Debug level log
func (z *ZapLogger) Debug(_ context.Context, args ...any) {
	z.Logger.Debug(fmt.Sprint(args...))
}

// Info build Info level log
func (z *ZapLogger) Info(_ context.Context, args ...any) {
	z.Logger.Info(fmt.Sprint(args...))
}

// Warn build Warn level log
func (z *ZapLogger) Warn(_ context.Context, args ...any) {
	z.Logger.Warn(fmt.Sprint(args...))
}

// Error build Error level log
func (z *ZapLogger) Error(_ context.Context, args ...any) {
	z.Logger.Error(fmt.Sprint(args...))
}
