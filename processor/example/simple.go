package example

import (
    "context"
    "time"
    "powerjob-client-go/processor"
)

// SimpleProcessor 一个示例处理器：等待指定毫秒并返回成功。
type SimpleProcessor struct{}

func (s *SimpleProcessor) Init(ctx context.Context) error { return nil }

func (s *SimpleProcessor) Run(ctx context.Context, params map[string]any) (processor.Result, error) {
    // 步骤：读取等待时长 -> 睡眠 -> 返回
    ms := int64(100)
    if v, ok := params["sleepMS"].(float64); ok { ms = int64(v) }
    select {
    case <-ctx.Done():
        return processor.Result{Code: -1, Msg: "canceled"}, ctx.Err()
    case <-time.After(time.Duration(ms) * time.Millisecond):
        return processor.Result{Code: 0, Msg: "ok"}, nil
    }
}

func (s *SimpleProcessor) Stop(ctx context.Context) error { return nil }

func init(){ processor.Register("simple", &SimpleProcessor{}) }

