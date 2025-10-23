package scheduler

import (
    "context"
    "time"

    "powerjob-client-go/client"
    "powerjob-client-go/logging"
)

// InstanceReporter 周期性上报实例状态。
// runningLister 只依赖 ListRunning，避免与具体存储强耦合。
// Running 最小化的运行中实例视图。
type Running struct { JobID int64; InstanceID int64; Status int }

// runningLister 仅需要列出运行中实例的精简信息。
type runningLister interface { ListRunning(ctx context.Context) ([]Running, error) }

type InstanceReporter struct {
    api     client.ServerAPI
    disc    *Discovery
    repo    runningLister
    worker  string
    interval time.Duration
}

// NewReporter 构造。
func NewReporter(api client.ServerAPI, disc *Discovery, repo runningLister, worker string, seconds int) *InstanceReporter {
    return &InstanceReporter{api: api, disc: disc, repo: repo, worker: worker, interval: time.Duration(seconds) * time.Second}
}

// Start 启动上报任务。
func (r *InstanceReporter) Start(ctx context.Context) {
    ticker := time.NewTicker(r.interval)
    go func() {
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                list, err := r.repo.ListRunning(ctx)
                if err != nil { logging.L().Warn(ctx, "list running failed", "err", err); continue }
                for _, it := range list {
                    req := client.TaskTrackerReportInstanceStatusReq{
                        JobID: it.JobID,
                        InstanceID: it.InstanceID,
                        ReportTime: time.Now().UnixMilli(),
                        SourceAddress: r.worker,
                        InstanceStatus: it.Status,
                    }
                    if err := r.api.ReportInstanceStatus(ctx, r.disc.Get(), req); err != nil { logging.L().Warn(ctx, "report instance failed", "iid", it.InstanceID, "err", err) }
                }
            }
        }
    }()
}
