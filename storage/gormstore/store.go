package gormstore

import (
    "context"
    "time"

    "gorm.io/gorm"
    "powerjob-client-go/powerjob"
)

// model 映射到数据库表。
type model struct {
    ID          uint      `gorm:"primaryKey"`
    InstanceID  int64     `gorm:"uniqueIndex"`
    JobID       int64     `gorm:"index"`
    Status      int       `gorm:"index"`
    Progress    int       `gorm:"default:0"`
    ResultCode  int       `gorm:"default:0"`
    ResultMsg   string    `gorm:"type:text"`
    StartedAt   time.Time `gorm:"index"`
    UpdatedAt   time.Time `gorm:"autoUpdateTime"`
}

// Store 基于 GORM 的 Storage 实现。
type Store struct { db *gorm.DB }

// New 创建 Store，调用方应自行在外部执行 AutoMigrate(&model{}).
func New(db *gorm.DB) *Store { return &Store{db: db} }

// Upsert 实现 Storage.Upsert。
func (s *Store) Upsert(ctx context.Context, rec *powerjob.InstanceRecord) error {
    m := toModel(rec)
    return s.db.WithContext(ctx).Where("instance_id = ?", rec.InstanceID).Assign(m).FirstOrCreate(&m).Error
}

// UpdateStatus 实现 Storage.UpdateStatus。
func (s *Store) UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error {
    return s.db.WithContext(ctx).Model(&model{}).
        Where("instance_id = ?", instanceID).
        Updates(map[string]any{"status": status, "result_code": resultCode, "result_msg": resultMsg}).Error
}

// Get 实现 Storage.Get。
func (s *Store) Get(ctx context.Context, instanceID int64) (*powerjob.InstanceRecord, error) {
    var m model
    if err := s.db.WithContext(ctx).Where("instance_id = ?", instanceID).First(&m).Error; err != nil {
        return nil, err
    }
    return fromModel(m), nil
}

// ListRunning 实现 Storage.ListRunning。
func (s *Store) ListRunning(ctx context.Context) ([]powerjob.InstanceRecord, error) {
    var list []model
    if err := s.db.WithContext(ctx).Where("status = ?", powerjob.StateRunning).Find(&list).Error; err != nil {
        return nil, err
    }
    out := make([]powerjob.InstanceRecord, 0, len(list))
    for _, m := range list { out = append(out, *fromModel(m)) }
    return out, nil
}

func toModel(r *powerjob.InstanceRecord) model {
    return model{ID: r.ID, InstanceID: r.InstanceID, JobID: r.JobID, Status: r.Status, Progress: r.Progress, ResultCode: r.ResultCode, ResultMsg: r.ResultMsg, StartedAt: r.StartedAt, UpdatedAt: r.UpdatedAt}
}

func fromModel(m model) *powerjob.InstanceRecord {
    return &powerjob.InstanceRecord{ID: m.ID, InstanceID: m.InstanceID, JobID: m.JobID, Status: m.Status, Progress: m.Progress, ResultCode: m.ResultCode, ResultMsg: m.ResultMsg, StartedAt: m.StartedAt, UpdatedAt: m.UpdatedAt}
}

