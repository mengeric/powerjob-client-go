package memstore

import (
    "context"
    "errors"
    "sync"
    "time"
    "github.com/mengeric/powerjob-client-go/powerjob"
)

// Store 是一个线程安全的内存实现，仅用于开发/轻量场景。
type Store struct {
    mu sync.RWMutex
    m  map[int64]*powerjob.InstanceRecord
}

// New 创建内存存储。
func New() *Store { return &Store{m: map[int64]*powerjob.InstanceRecord{}} }

func (s *Store) Upsert(ctx context.Context, rec *powerjob.InstanceRecord) error {
    s.mu.Lock(); defer s.mu.Unlock()
    cp := *rec
    if old, ok := s.m[rec.InstanceID]; ok {
        cp.ID = old.ID
    }
    if cp.UpdatedAt.IsZero() { cp.UpdatedAt = time.Now() }
    s.m[rec.InstanceID] = &cp
    return nil
}

func (s *Store) UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error {
    s.mu.Lock(); defer s.mu.Unlock()
    if r, ok := s.m[instanceID]; ok {
        r.Status = status
        r.ResultCode = resultCode
        r.ResultMsg = resultMsg
        r.UpdatedAt = time.Now()
        return nil
    }
    return errors.New("not found")
}

func (s *Store) Get(ctx context.Context, instanceID int64) (*powerjob.InstanceRecord, error) {
    s.mu.RLock(); defer s.mu.RUnlock()
    if r, ok := s.m[instanceID]; ok {
        cp := *r
        return &cp, nil
    }
    return nil, errors.New("not found")
}

func (s *Store) ListRunning(ctx context.Context) ([]powerjob.InstanceRecord, error) {
    s.mu.RLock(); defer s.mu.RUnlock()
    out := make([]powerjob.InstanceRecord, 0)
    for _, v := range s.m {
        if v.Status == powerjob.StateRunning {
            out = append(out, *v)
        }
    }
    return out, nil
}

