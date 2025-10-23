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
// 功能：执行业务逻辑；Stop 用于响应停止；GetTaskKey 返回控制台配置的 processorInfo。
// 约定：GetTaskKey() 的返回值必须与控制台中的 processorInfo 完全一致（区分大小写）。
type Processor interface {
    // GetTaskKey 返回该处理器的唯一键（即控制台 processorInfo）。
    GetTaskKey() string
    // Init 初始化钩子，可选。
    Init(ctx context.Context) error
    // Run 执行业务，raw 为原始 JSON 字节，请自行绑定到强类型。
    Run(ctx context.Context, raw []byte) (Result, error)
    // Stop 停止钩子，接收取消通知后进行清理。
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
// 功能：使用 p.GetTaskKey() 作为键，不再需要额外传入名称，避免人为不一致。
// 注意：若 key 为空将 panic；重复注册将覆盖旧值（请自行避免）。
func Register(p Processor) {
    key := p.GetTaskKey()
    if key == "" {
        panic("processor.GetTaskKey() must not be empty")
    }
    regMu.Lock()
    processors[key] = p
    regMu.Unlock()
}

// Get 获取处理器。
func Get(name string) (Processor, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	p, ok := processors[name]
	return p, ok
}

// ErrNotFound 处理器不存在错误。
var ErrNotFound = errors.New("processor not found")

// RegisterTyped 已移除：请直接实现 Processor + GetTaskKey，在 Run 中自行解码 raw。
func RegisterTyped[S any](name string, p TypedProcessor[S]) { panic("TypedProcessor removed; implement Processor + GetTaskKey and decode raw yourself") }

// typedAdapter 将 TypedProcessor[S] 适配为旧 Processor 接口，便于 Worker 统一调用路径。
type typedAdapter[S any] struct{}
