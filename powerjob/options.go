package powerjob

import (
	"github.com/mengeric/powerjob-client-go/client"
	"time"
)

// Options 组件运行参数。
// 功能：描述与 PowerJob-Server 的交互周期、监听端口、在线日志等行为；
// 说明：组件会在 Start(ctx) 内部启动内置 HTTP Server（监听 ListenAddr）。
type Options struct {
	ListenAddr      string        // HTTP 服务监听地址，例如 :27777 或 127.0.0.1:0（0 表示随机端口）
	BootstrapServer string        // 引导地址，如 127.0.0.1:7700
	AppName         string        // 应用名
	ClientVersion   string        // 客户端版本
	HeartbeatEvery  time.Duration // 心跳上报周期
	ReportEvery     time.Duration // 实例状态上报周期
	DiscoveryEvery  time.Duration // 服务发现刷新周期
	WorkerAddress   string        // 向 Server 上报的 workerAddress（外部可见地址）
	LogReportEvery  time.Duration // 在线日志上报周期
	LogBatchSize    int           // 在线日志单批最大条数
}

// withDefaults 填充默认值。
func (o *Options) withDefaults() {
	if o.ListenAddr == "" {
		o.ListenAddr = ":27777"
	}
	if o.HeartbeatEvery <= 0 {
		o.HeartbeatEvery = 15 * time.Second
	}
	if o.ReportEvery <= 0 {
		o.ReportEvery = 10 * time.Second
	}
	if o.DiscoveryEvery <= 0 {
		o.DiscoveryEvery = 30 * time.Second
	}
	if o.LogReportEvery <= 0 {
		o.LogReportEvery = 10 * time.Second
	}
	if o.LogBatchSize <= 0 {
		o.LogBatchSize = 256
	}
}

// Option 函数式可选项，用于构造 Worker。
type Option func(*workerConfig)

type workerConfig struct {
    opt   Options
    store Storage
    api   client.ServerAPI
}

// WithOptions 批量设置运行参数。
func WithOptions(o Options) Option { return func(c *workerConfig) { c.opt = o } }

// WithListenAddr 设置监听地址。
func WithListenAddr(addr string) Option { return func(c *workerConfig) { c.opt.ListenAddr = addr } }

// WithBootstrapServer 设置引导地址。
func WithBootstrapServer(s string) Option { return func(c *workerConfig) { c.opt.BootstrapServer = s } }

// WithAppName 设置应用名。
func WithAppName(s string) Option { return func(c *workerConfig) { c.opt.AppName = s } }

// WithClientVersion 设置客户端版本。
func WithClientVersion(s string) Option { return func(c *workerConfig) { c.opt.ClientVersion = s } }

// WithWorkerAddress 设置上报给 Server 的 workerAddress。
func WithWorkerAddress(s string) Option { return func(c *workerConfig) { c.opt.WorkerAddress = s } }

// WithIntervals 快速设置心跳/上报/发现周期（可选）。
func WithIntervals(hb, rep, disc time.Duration) Option {
	return func(c *workerConfig) { c.opt.HeartbeatEvery, c.opt.ReportEvery, c.opt.DiscoveryEvery = hb, rep, disc }
}

// WithLogReporter 配置在线日志上报参数。
func WithLogReporter(every time.Duration, batch int) Option {
	return func(c *workerConfig) { c.opt.LogReportEvery, c.opt.LogBatchSize = every, batch }
}

// withStore 仅测试或高级接入使用：替换默认内存存储。
func withStore(s Storage) Option { return func(c *workerConfig) { c.store = s } }

// WithClientAPI 替换默认 ServerAPI（测试场景使用）。
func WithClientAPI(api client.ServerAPI) Option { return func(c *workerConfig) { c.api = api } }
