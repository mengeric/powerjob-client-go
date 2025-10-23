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
    Run(ctx context.Context, raw []byte) (Result, error)
    Stop(ctx context.Context) error
}

// TypedProcessor 是基于泛型的强类型处理器接口。
// 功能：直接以强类型 S 接收参数，避免 map 解析与断言。
// 使用：通过 RegisterTyped(name, impl) 注册，组件会负责把原始 JSON 绑定为 S。
type TypedProcessor[S any] interface{}

var (
	regMu      sync.RWMutex
	processors = map[string]Processor{}
)

// Register 注册处理器。
func Register(name string, p Processor) { regMu.Lock(); defer regMu.Unlock(); processors[name] = p }

// Get 获取处理器。
func Get(name string) (Processor, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	p, ok := processors[name]
	return p, ok
}

// ErrNotFound 处理器不存在错误。
var ErrNotFound = errors.New("processor not found")

// RegisterTyped 使用泛型注册强类型处理器，内部自动完成 JSON → S 绑定。
func RegisterTyped[S any](name string, p TypedProcessor[S]) { panic("TypedProcessor removed; use Processor with []byte and decode yourself") }

// typedAdapter 将 TypedProcessor[S] 适配为旧 Processor 接口，便于 Worker 统一调用路径。
type typedAdapter[S any] struct{}
