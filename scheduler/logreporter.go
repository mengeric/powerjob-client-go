package scheduler

import (
	"context"
	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/logging"
	"time"
)

// LogReporter 批量上报在线日志。
type LogReporter struct {
	api    client.ServerAPI
	disc   *Discovery
	worker string
	ch     chan client.InstanceLogContent
	tick   time.Duration
	max    int
}

// NewLogReporter 创建日志上报器。
// 参数：intervalSeconds 上报周期（秒）；batchMax 单批最大条数。
func NewLogReporter(api client.ServerAPI, disc *Discovery, worker string, intervalSeconds, batchMax int) *LogReporter {
	if batchMax <= 0 {
		batchMax = 256
	}
	lr := &LogReporter{
		api:    api,
		disc:   disc,
		worker: worker,
		ch:     make(chan client.InstanceLogContent, batchMax*4),
		tick:   time.Duration(intervalSeconds) * time.Second,
		max:    batchMax,
	}
	return lr
}

// Start 启动后台上报协程。
func (l *LogReporter) Start(ctx context.Context) {
	ticker := time.NewTicker(l.tick)
	go func() {
		defer ticker.Stop()
		buf := make([]client.InstanceLogContent, 0, l.max)
		flush := func() {
			if len(buf) == 0 {
				return
			}
			req := client.WorkerLogReportReq{InstanceLogContents: buf, WorkerAddress: l.worker}
            if err := l.api.ReportLog(ctx, l.disc.Get(), req); err != nil {
                logging.L().Warnf(ctx, "report log failed: count=%d err=%v", len(buf), err)
            }
			buf = buf[:0]
		}
		for {
			select {
			case <-ctx.Done():
				flush()
				return
			case it := <-l.ch:
				if it.InstanceID == 0 || it.LogTime == 0 {
					continue
				}
				buf = append(buf, it)
				if len(buf) >= l.max {
					flush()
				}
			case <-ticker.C:
				flush()
			}
		}
	}()
}

// Enqueue 推入一条日志（非阻塞，满了会丢弃并告警）。
func (l *LogReporter) Enqueue(it client.InstanceLogContent) {
	select {
	case l.ch <- it:
	default:
        logging.L().Warnf(context.Background(), "log queue full, drop: iid=%d", it.InstanceID)
	}
}
