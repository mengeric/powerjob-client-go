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
    ID         uint
    InstanceID int64
    JobID      int64
    Status     int
    Progress   int
    ResultCode int
    ResultMsg  string
    StartedAt  time.Time
    UpdatedAt  time.Time
}

// Storage 为最小持久化接口。
// 说明：当前版本默认使用内置内存实现；外部可按需实现替换（文档默认不暴露）。
type Storage interface {
    Upsert(ctx context.Context, rec *InstanceRecord) error
    UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error
    Get(ctx context.Context, instanceID int64) (*InstanceRecord, error)
    ListRunning(ctx context.Context) ([]InstanceRecord, error)
}
