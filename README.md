powerjob-client-go（组件版）
================================

面向 PowerJob 的 Go 语言 Worker 组件库。它不是一个 Web 工程，也不内置 HTTP 服务器；而是以“组件模式”输出：你在宿主服务中注入配置和存储，实现/注册业务处理器，挂载组件提供的 HTTP Handler，即可接入 PowerJob 调度。

- 语言/版本：Go 1.24.3
- 依赖：标准库 + GORM 可选（用于示例存储实现）
- 日志：提供简单日志门面，可替换

一、架构概览
------------

组件分层如下（省略非关键细节）：

- powerjob
  - Options：运行选项（与 Server 交互周期、workerAddress 等）
  - Storage：持久化接口（Upsert/UpdateStatus/Get/ListRunning）
  - Worker：组件主体，提供
    - MountHTTP(mux, base)：把 /worker/* Handler 挂载到宿主路由
    - Start(ctx)：启动服务发现/心跳/实例状态上报等后台循环
- internal
  - client：与 PowerJob-Server 通讯（/server/assert、/server/acquire、心跳与上报）
  - scheduler：服务发现、心跳、实例状态上报定时任务
  - processor：业务处理器接口与注册表（示例 simple）
  - worker/tracker：本地实例生命周期跟踪
  - logging：日志门面（默认 slog，可替换）
- storage
  - gormstore：基于 GORM 的 Storage 实现（可选）

数据流（简化）：
- Server → Worker：POST /worker/runJob 触发本地 Processor 执行；/worker/stopInstance 请求停止；/worker/queryInstanceStatus 查询状态
- Worker → Server：定时 /server/workerHeartbeat 与 /server/reportInstanceStatus，上报心跳与实例状态
- Worker ↔ Store：实例入库、状态迁移、查询运行中实例

二、目录结构
------------

```
.
├─ powerjob/
│  ├─ options.go           # Options 运行参数
│  ├─ storage.go           # Storage 接口与状态常量
│  └─ worker.go            # 组件主体（HTTP Handler + 后台调度）
├─ client/                 # 与 Server 的 HTTP 通讯
├─ scheduler/              # 服务发现/心跳/上报
├─ logging/                # 日志门面
├─ processor/              # 处理器接口与示例
├─ tracker/                # 实例跟踪器
├─ storage/gormstore/      # 可选：GORM 持久化实现
├─ etc/worker.yaml         # 示例配置（仅参考）
├─ Makefile                # build/test/mocks
└─ README.md
```

三、快速开始（集成到你的宿主服务）
----------------------------

1) 安装依赖

```
go get gorm.io/gorm gorm.io/driver/mysql
```

2) （可选）注册你的业务处理器

```go
package myproc

import (
    "context"
    "github.com/mengeric/powerjob-client-go/processor"
)

// DemoProcessor 示例：等待一段时间并返回成功
type DemoProcessor struct{}

func (d *DemoProcessor) Init(ctx context.Context) error { return nil }
func (d *DemoProcessor) Run(ctx context.Context, params map[string]any) (processor.Result, error) {
    // 读取参数并执行业务……
    return processor.Result{Code: 0, Msg: "ok"}, nil
}
func (d *DemoProcessor) Stop(ctx context.Context) error { return nil }

// 包级 init：正确的自动注册方式（注意：必须是包级，而不是方法）
func init() { processor.Register("demo", &DemoProcessor{}) }
```

3) 实现/选择 Storage（两种方式）

- 方式 A：使用内置 GORM 存储

```go
import (
    "gorm.io/gorm"
    "gorm.io/driver/mysql"
    "github.com/mengeric/powerjob-client-go/storage/gormstore"
)

// 初始化 GORM
dsn := "user:pass@tcp(127.0.0.1:3306)/powerjob_go?charset=utf8mb4&parseTime=true&loc=Local"
db, _ := gorm.Open(mysql.Open(dsn), &gorm.Config{})
// 迁移表结构（建议复制 gormstore 的 model 字段创建同构结构）
db.AutoMigrate(&struct{ /* 建议复制 gormstore 的 model 字段 */ }{})
store := gormstore.New(db)
```

- 方式 B：实现自定义存储（示例）

```go
type MyStore struct{}
func (s *MyStore) Upsert(ctx context.Context, rec *powerjob.InstanceRecord) error { /*...*/ return nil }
func (s *MyStore) UpdateStatus(ctx context.Context, id int64, st int, code int, msg string) error { /*...*/ return nil }
func (s *MyStore) Get(ctx context.Context, id int64) (*powerjob.InstanceRecord, error) { /*...*/ return nil, nil }
func (s *MyStore) ListRunning(ctx context.Context) ([]powerjob.InstanceRecord, error) { /*...*/ return nil, nil }
```

4) 组装 Worker 并挂载到宿主 HTTP Server

```go
import (
    "context"
    "net/http"
    "time"
    "github.com/mengeric/powerjob-client-go/powerjob"
)

store := /* 见上 */
opt := powerjob.Options{
    BootstrapServer:  "127.0.0.1:7700",
    AppName:          "demo",
    ClientVersion:    "0.1.0",
    WorkerAddress:    "10.0.0.5:27777", // Server 可访问到的对外地址
    HeartbeatEvery:   15 * time.Second,
    ReportEvery:      10 * time.Second,
    DiscoveryEvery:   30 * time.Second,
}

w := powerjob.NewWorker(store, opt, nil) // 第三个参数为自定义 ServerAPI，nil 使用默认 HTTP 实现

// 后台任务（服务发现/心跳/上报）
ctx, cancel := context.WithCancel(context.Background())
go w.Start(ctx)
defer cancel()

// 挂载组件提供的 Handler
mux := http.NewServeMux()
w.MountHTTP(mux, "/worker")

http.ListenAndServe(":27777", mux)
```

> 提示：组件不持有或创建 HTTP Server，端口与生命周期完全由宿主控制。

四、HTTP 接口（由组件提供）
----------------------

- POST /worker/runJob
  - 请求体：ServerScheduleJobReq（参见 client/types.go）
  - 行为：入库运行中 → 异步执行 Processor → 状态迁移
- POST /worker/stopInstance
  - 请求体：{"instanceId": 123}
  - 行为：取消本地实例执行，状态置为 Stopped
- POST /worker/queryInstanceStatus
  - 请求体：{"instanceId": 123}
  - 返回：InstanceRecord（powerjob.Storage 模型）

五、Options 字段说明
----------------

- BootstrapServer：Server 引导地址（用于 /server/assert 与 /server/acquire）
- AppName / ClientVersion：应用标识
- WorkerAddress：上报给 Server 的可访问地址（通常为公网或内网可路由地址）
- HeartbeatEvery / ReportEvery / DiscoveryEvery：心跳/状态上报/服务发现周期，留空采用默认值 15s/10s/30s

六、Storage 接口
--------------

```go
type Storage interface {
    Upsert(ctx context.Context, rec *powerjob.InstanceRecord) error
    UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error
    Get(ctx context.Context, instanceID int64) (*powerjob.InstanceRecord, error)
    ListRunning(ctx context.Context) ([]powerjob.InstanceRecord, error)
}
```

> 组件仅依赖这些最小能力；你可以替换为任意实现（例如另外的 ORM/DAO/远程存储）。

七、日志门面
----------

- 包路径：logging
- 默认实现：slog 文本输出
- 自定义注入：`logging.SetGlobal(myLogger)`，实现 Logger 接口（Info/Warn/Error/Debug/With）

八、处理器开发规范
--------------

- 接口：`processor.Processor`
  - `Init(ctx)`—可选初始化；`Run(ctx, params)`—执行；`Stop(ctx)`—响应停止
- 注册：`processor.Register("yourName", yourProcessor)`，控制台中的 `processorInfo` 与此名称对应
- 参数：`jobParams` 为字符串 JSON，组件会解成 `map[string]any`
- 幂等：同一 `instanceId` 触发的重复 `runJob` 将直接返回 200（已在执行/已完成）

九、测试与构建
------------

- 运行测试：`make test`（已启用 `-race -cover`）
- 生成 gomock 桩：`make mock`
- 覆盖率摘要：`make cover`

> 示例测试使用 GoConvey BDD 风格（powerjob/worker_http_test.go），可按该风格扩展到你的处理器与存储实现。

十、生产建议
----------

- WorkerAddress 要求 Server 可达（NAT/网关需配置端口映射或反向代理）
- 存储层需确保状态迁移的事务一致性与幂等
- 大量日志/上报建议在存储或网络层做限频与分批
- 任务执行请尊重 ctx 取消以支持停止操作

十一、兼容性与限制
--------------

- Go 1.24.3、GORM v2
- 组件不依赖 go-zero，不提供内置 Web Server
- MAP/MAP_REDUCE 的 worker 端细节需按你的业务在处理器内部实现（协议层不限制）

——

如需英文 README、内存存储实现或覆盖率 ≥80% 的补充测试，请在 Issue 中告知或直接提出需求。
