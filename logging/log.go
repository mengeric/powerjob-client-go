package logging

import (
    "context"
    "log/slog"
    "os"
)

// Logger 日志门面接口。
// 说明：为了最小侵入，提供 Info/Warn/Error/Debug 与 With 方法。
type Logger interface {
    Info(ctx context.Context, msg string, args ...any)
    Warn(ctx context.Context, msg string, args ...any)
    Error(ctx context.Context, msg string, args ...any)
    Debug(ctx context.Context, msg string, args ...any)
    With(args ...any) Logger
}

// SlogLogger 基于标准库 slog 的默认实现。
type SlogLogger struct{ l *slog.Logger }

// NewSlogLogger 创建默认 slog 日志器（文本输出）。
func NewSlogLogger() *SlogLogger {
    return &SlogLogger{l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))}
}

// SetLevel 设置日志级别。
func (s *SlogLogger) SetLevel(level slog.Level) { s.l = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})) }

func (s *SlogLogger) Info(ctx context.Context, msg string, args ...any)  { s.l.InfoContext(ctx, msg, args...) }
func (s *SlogLogger) Warn(ctx context.Context, msg string, args ...any)  { s.l.WarnContext(ctx, msg, args...) }
func (s *SlogLogger) Error(ctx context.Context, msg string, args ...any) { s.l.ErrorContext(ctx, msg, args...) }
func (s *SlogLogger) Debug(ctx context.Context, msg string, args ...any) { s.l.DebugContext(ctx, msg, args...) }
func (s *SlogLogger) With(args ...any) Logger                          { return &SlogLogger{l: s.l.With(args...)} }

// 全局默认日志器，便于简化调用。
var defaultLogger Logger = NewSlogLogger()

// L 获取全局日志器。
func L() Logger { return defaultLogger }

// SetGlobal 替换全局日志器（如业务侧注入第三方实现）。
func SetGlobal(l Logger) { if l != nil { defaultLogger = l } }

