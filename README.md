powerjob-client-go（组件版）
================================

面向 PowerJob 的 Go 语言 Worker 组件库。以“组件模式”接入：在宿主服务中创建 `Worker`、设置运行选项并注册处理器，调用 `Start(ctx)` 即可启用。组件会在内部自动启动 HTTP Server（默认 `:27777`），无需挂载任何 Handler。

- 语言/版本：Go 1.24.3
- 依赖：标准库（默认内存存储，无数据库强依赖）
- 日志：内置门面（基于 slog），支持在线日志自动上报

一、项目介绍
------------
- 定位：与 PowerJob-Server 对接的 Go Worker 组件，专注“拉起即可用”的最小稳定集（服务发现/心跳/实例状态/在线日志）。
- 形态：无 main、无 Web 框架依赖；仅暴露 `powerjob.NewWorker(...).Start(ctx)`。
- 目标：简单、可测试、可观测；遵循 SOLID/DRY/KISS/YAGNI。

二、接入说明
------------
1) 安装与导入
```
go get github.com/mengeric/powerjob-client-go
```
```
import (
  "github.com/mengeric/powerjob-client-go/powerjob"
  "github.com/mengeric/powerjob-client-go/processor"
  "github.com/mengeric/powerjob-client-go/logging"
)
```

2) 注册处理器（包级 init；GetTaskKey 必须与控制台 processorInfo 严格一致）
```go
package myproc

import (
  "context"
  "github.com/mengeric/powerjob-client-go/processor"
)

type DemoProcessor struct{}
func (d *DemoProcessor) GetTaskKey() string { return "demo" }
func (d *DemoProcessor) Init(ctx context.Context) error { return nil }
func (d *DemoProcessor) Run(ctx context.Context, raw []byte) (processor.Result, error) {
  // 在此自行 json.Unmarshal(raw, &YourParams)
  return processor.Result{Code:0, Msg:"ok"}, nil
}
func (d *DemoProcessor) Stop(ctx context.Context) error { return nil }

func init() { processor.Register(&DemoProcessor{}) }
```

3) 启动 Worker（内置 HTTP 自动启动）
```go
ctx := context.Background()
w := powerjob.NewWorker(
  powerjob.WithBootstrapServer("127.0.0.1:7700"),
  powerjob.WithAppName("demo"),
  powerjob.WithClientVersion("0.1.0"),
  powerjob.WithListenAddr(":27777"), // 可用 ":0" 随机端口
  powerjob.WithIntervals(15*time.Second, 10*time.Second, 30*time.Second),
  powerjob.WithLogReporter(10*time.Second, 256),
)
go w.Start(ctx)
println("listening:", w.Addr())
```

4) 优雅关闭（可选）
```go
base := context.Background()
ctx, stop := powerjob.WithSignalCancel(base)
defer stop()
w := powerjob.NewWorker(powerjob.WithBootstrapServer("127.0.0.1:7700"), powerjob.WithAppName("demo"))
go w.Start(ctx)
<-ctx.Done()
```

5) 在线日志（推荐）
- 在处理器中传递组件提供的 `ctx` 给日志门面：
```go
logging.L().Infof(ctx, "start iid=%d", 123)
```
- 组件会把带实例上下文的日志自动批量上报；非处理器处可用 `w.Log(instanceID, level, content, timeMs)`。

三、参数项（Options）
------------------
- `ListenAddr`：HTTP 监听地址，默认 `:27777`；支持 `:0` 随机端口（用 `w.Addr()` 获取实际端口）。
- `BootstrapServer`：引导地址（用于 assert/acquire）。
- `AppName`、`ClientVersion`：应用标识。
- `WorkerAddress`：上报给 Server 的可访问地址；留空则使用实际监听地址。
- `HeartbeatEvery`、`ReportEvery`、`DiscoveryEvery`：心跳/状态/发现周期，默认 15s/10s/30s。
- `LogReportEvery`、`LogBatchSize`：在线日志上报周期与单批大小，默认 10s/256。

四、最佳实践
------------
- 名称对齐：处理器 `GetTaskKey()` 与控制台 `processorInfo` 严格一致（区分大小写）。
```go 
package myproc

import "github.com/mengeric/powerjob-client-go/processor"

// 控制台填写的 processorInfo = "order.settle.v1"
func (p *OrderSettleV1) GetTaskKey() string { return "order.settle.v1" }
func init() { processor.Register(&OrderSettleV1{}) }
```

- 参数绑定：处理器签名为 `Run(ctx, raw []byte)`，直接 `json.Unmarshal(raw, &YourParams)` 完成强类型绑定；不做 `map[string]any` 中转。
```go
type SettleParams struct {
  OrderID string `json:"orderId"`
  Force   bool   `json:"force"`
}

type OrderSettleV1 struct{}

// Run 执行业务：使用强类型参数避免魔法 map
func (p *OrderSettleV1) Run(ctx context.Context, raw []byte) (processor.Result, error) {
  var in SettleParams
  if err := json.Unmarshal(raw, &in); err != nil {
    logging.L().Errorf(ctx, "param decode failed: %v", err)
    return processor.Result{Code: -1, Msg: "bad params"}, err
  }
  // ...执行业务...
  return processor.Result{Code: 0, Msg: "ok"}, nil
}
```

- 日志上报：处理器内使用 `logging.L().Infof(ctx, ...)`，组件自动上报；非处理器使用 `w.Log(...)` 手动上报。
```go
// 自动上报（推荐）：ctx 带有实例上下文，将被组件 Hook 捕获并上报
logging.L().Infof(ctx, "settle start orderId=%s", in.OrderID)

// 手动上报（可选）：在非处理器位置
w.Log(instanceID, 2 /*INFO*/, "background progress...", 0)
```

- 端口与地址：默认回填 `WorkerAddress=实际监听地址`；若在容器/NAT 场景，显式设置可达地址（如 Host 端口映射/反代）。
```go
w := powerjob.NewWorker(
  powerjob.WithListenAddr(":27777"),              // 容器内监听
  powerjob.WithWorkerAddress("host.example.com:80"), // Server 可达地址（经反代）
  powerjob.WithBootstrapServer("server:7700"),
  powerjob.WithAppName("demo"),
)
```

- 任务可取消：处理器务必尊重 `ctx.Done()`，便于 `stopInstance` 生效并快速回收资源。
```go
func (p *OrderSettleV1) Run(ctx context.Context, raw []byte) (processor.Result, error) {
  ticker := time.NewTicker(200 * time.Millisecond)
  defer ticker.Stop()
  for step := 0; step < 10; step++ {
    select {
    case <-ctx.Done():
      return processor.Result{Code: 0, Msg: "canceled"}, ctx.Err()
    case <-ticker.C:
      // do partial work...
    }
  }
  return processor.Result{Code: 0, Msg: "ok"}, nil
}
```

- 测试建议：使用 GoConvey + gomock；覆盖率≥80%，关注错误路径与重试。
```go
// GoConvey 样例（断言与分组）
func TestSettle_Run(t *testing.T) {
  Convey("settle ok", t, func() {
    p := &OrderSettleV1{}
    raw := []byte(`{"orderId":"A1","force":true}`)
    got, err := p.Run(context.Background(), raw)
    So(err, ShouldBeNil)
    So(got.Code, ShouldEqual, 0)
  })
}

// gomock 样例（打桩 ServerAPI）
ctrl := gomock.NewController(t)
defer ctrl.Finish()
m := mocks.NewMockServerAPI(ctrl)
m.EXPECT().ReportInstanceStatus(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
```
