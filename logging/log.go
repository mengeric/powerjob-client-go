package logging

import (
    "context"
    "fmt"
    "log/slog"
    "os"
    "sync/atomic"
)

// Logger 日志门面接口（fmt 风格）。
// 说明：全部采用 *f 方法（Infof/Warnf/Errorf/Debugf），第一个参数为 format，后续为格式化参数。
type Logger interface {
    // Infof 以 Info 级别输出格式化日志。
    // 参数：ctx 上下文；format 格式串；args 格式化参数。
    Infof(ctx context.Context, format string, args ...any)
    // Warnf 以 Warn 级别输出格式化日志。
    Warnf(ctx context.Context, format string, args ...any)
    // Errorf 以 Error 级别输出格式化日志。
    Errorf(ctx context.Context, format string, args ...any)
    // Debugf 以 Debug 级别输出格式化日志。
    Debugf(ctx context.Context, format string, args ...any)
    // With 预留结构化扩展，不在本版本使用。
    With(args ...any) Logger
}

// SlogLogger 基于标准库 slog 的默认实现。
type SlogLogger struct{ l *slog.Logger }

// NewSlogLogger 创建默认 slog 日志器（文本输出）。
func NewSlogLogger() *SlogLogger {
    return &SlogLogger{l: slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))}
}

// SetLevel 设置日志级别。
func (s *SlogLogger) SetLevel(level slog.Level) {
    s.l = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level}))
}

// Infof 输出 Info 日志（fmt 风格），并触发 Hook。
func (s *SlogLogger) Infof(ctx context.Context, format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    s.l.InfoContext(ctx, msg)
    callHook(ctx, 2, msg)
}

// Warnf 输出 Warn 日志（fmt 风格），并触发 Hook。
func (s *SlogLogger) Warnf(ctx context.Context, format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    s.l.WarnContext(ctx, msg)
    callHook(ctx, 3, msg)
}

// Errorf 输出 Error 日志（fmt 风格），并触发 Hook。
func (s *SlogLogger) Errorf(ctx context.Context, format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    s.l.ErrorContext(ctx, msg)
    callHook(ctx, 4, msg)
}

// Debugf 输出 Debug 日志（fmt 风格），并触发 Hook。
func (s *SlogLogger) Debugf(ctx context.Context, format string, args ...any) {
    msg := fmt.Sprintf(format, args...)
    s.l.DebugContext(ctx, msg)
    callHook(ctx, 1, msg)
}

// With 预留（返回共享同一 slog 实例的包装）。
func (s *SlogLogger) With(args ...any) Logger { return &SlogLogger{l: s.l.With(args...)} }

// 全局默认日志器。
var defaultLogger Logger = NewSlogLogger()

// L 获取全局日志器。
func L() Logger { return defaultLogger }

// SetGlobal 替换全局日志器（如业务侧注入第三方实现）。
func SetGlobal(l Logger) {
    if l != nil {
        defaultLogger = l
    }
}

// Hook 用于拦截日志并执行附加行为（例如在线日志上报）。
// 注意：Hook 内部不应再次调用 logging.L() 以避免递归。
type Hook func(ctx context.Context, level int, msg string, args ...any)

var hook atomic.Value // stores Hook

// SetHook 设置全局日志 Hook；传入 nil 表示移除 Hook。
func SetHook(h Hook) {
    if h == nil { hook.Store(nil); return }
    hook.Store(h)
}

func callHook(ctx context.Context, level int, msg string, args ...any) {
    if v := hook.Load(); v != nil {
        v.(Hook)(ctx, level, msg, args...)
    }
}
