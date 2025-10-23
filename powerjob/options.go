package powerjob

import "time"

// Options 组件运行参数。
// 功能：描述与 PowerJob-Server 的交互周期等行为；不含任何 Web 框架配置。
// 注意：宿主服务自行决定监听端口；本组件仅提供 HTTP Handler 挂载能力。
type Options struct {
    BootstrapServer  string        // 引导地址，如 127.0.0.1:7700
    AppName          string        // 应用名
    ClientVersion    string        // 客户端版本
    HeartbeatEvery   time.Duration // 心跳上报周期
    ReportEvery      time.Duration // 实例状态上报周期
    DiscoveryEvery   time.Duration // 服务发现刷新周期
    WorkerAddress    string        // 向 Server 上报的 workerAddress（外部可见地址）
}

// withDefaults 填充默认值。
func (o *Options) withDefaults() {
    if o.HeartbeatEvery <= 0 { o.HeartbeatEvery = 15 * time.Second }
    if o.ReportEvery <= 0 { o.ReportEvery = 10 * time.Second }
    if o.DiscoveryEvery <= 0 { o.DiscoveryEvery = 30 * time.Second }
}

