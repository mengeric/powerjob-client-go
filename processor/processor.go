package processor

import (
    "context"
    "errors"
    "sync"
)

// Result 处理器执行结果。
type Result struct {
    Code int
    Msg  string
}

// Processor 统一处理器接口。
// 功能：执行业务逻辑；Stop 用于响应停止。
type Processor interface {
    Init(ctx context.Context) error
    Run(ctx context.Context, params map[string]any) (Result, error)
    Stop(ctx context.Context) error
}

var (
    regMu     sync.RWMutex
    processors = map[string]Processor{}
)

// Register 注册处理器。
func Register(name string, p Processor) { regMu.Lock(); defer regMu.Unlock(); processors[name] = p }

// Get 获取处理器。
func Get(name string) (Processor, bool) { regMu.RLock(); defer regMu.RUnlock(); p, ok := processors[name]; return p, ok }

// ErrNotFound 处理器不存在错误。
var ErrNotFound = errors.New("processor not found")

