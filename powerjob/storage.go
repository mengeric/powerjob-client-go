package powerjob

import (
    "context"
    "time"
)

// 实例状态常量，保持与多语言文档一致。
const (
    StateWaitingDispatch      = 1
    StateWaitingWorkerReceive = 2
    StateRunning              = 3
    StateFailed               = 4
    StateSucceed              = 5
    StateCanceled             = 9
    StateStopped              = 10
)

// InstanceRecord 任务实例持久化实体（最小字段集）。
type InstanceRecord struct {
    ID          uint
    InstanceID  int64
    JobID       int64
    Status      int
    Progress    int
    ResultCode  int
    ResultMsg   string
    StartedAt   time.Time
    UpdatedAt   time.Time
}

// Storage 持久化接口（可由宿主实现或使用内置 gormstore）。
type Storage interface {
    // Upsert 插入或更新实例记录。
    Upsert(ctx context.Context, rec *InstanceRecord) error
    // UpdateStatus 更新实例状态与结果信息。
    UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error
    // Get 按 instanceID 获取记录。
    Get(ctx context.Context, instanceID int64) (*InstanceRecord, error)
    // ListRunning 列出运行中实例。
    ListRunning(ctx context.Context) ([]InstanceRecord, error)
}

