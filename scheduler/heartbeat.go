package scheduler

import (
	"context"
	"runtime"
	"time"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/logging"
	"github.com/mengeric/powerjob-client-go/metrics"
)

// HeartbeatScheduler 周期性上报心跳。
type HeartbeatScheduler struct {
	api        client.ServerAPI
	disc       *Discovery
	workerAddr string
	interval   time.Duration
}

// NewHeartbeat 构造。
func NewHeartbeat(api client.ServerAPI, disc *Discovery, workerAddr string, seconds int) *HeartbeatScheduler {
	return &HeartbeatScheduler{api: api, disc: disc, workerAddr: workerAddr, interval: time.Duration(seconds) * time.Second}
}

// Start 启动心跳。
func (h *HeartbeatScheduler) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				_ = runtime.NumGoroutine() // 留作示例：如需纳入评分，可在 metrics 中扩展
				hb := client.WorkerHeartbeat{
					WorkerAddress: h.workerAddr,
					HeartbeatTime: time.Now().UnixMilli(),
					Protocol:      "HTTP",
					SystemMetrics: metrics.CollectSystemMetric(ctx),
				}
				if err := h.api.Heartbeat(ctx, h.disc.Get(), hb); err != nil {
					logging.L().Warn(ctx, "heartbeat failed", "err", err)
				}
			}
		}
	}()
}
