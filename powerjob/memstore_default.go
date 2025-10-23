package powerjob

import (
	"context"
	"errors"
	"sync"
	"time"
)

// inMemoryStore 是包内置的线程安全内存存储，仅用于默认与测试场景。
// 设计：为了避免 import cycle，不依赖外部子包，实现最小的 Storage 接口。
type inMemoryStore struct {
	mu sync.RWMutex
	m  map[int64]*InstanceRecord
}

// newDefaultMemStore 创建内置内存存储实现。
// 返回：满足 Storage 的实现。
func newDefaultMemStore() Storage { return &inMemoryStore{m: map[int64]*InstanceRecord{}} }

// Upsert 插入或更新实例记录。
// 参数：
// - rec：实例记录指针；若不存在则插入，存在则按 instanceId 覆盖；
// 返回：错误信息；正常情况返回 nil。
func (s *inMemoryStore) Upsert(ctx context.Context, rec *InstanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	cp := *rec
	if old, ok := s.m[rec.InstanceID]; ok {
		cp.ID = old.ID
	}
	if cp.UpdatedAt.IsZero() {
		cp.UpdatedAt = time.Now()
	}
	s.m[rec.InstanceID] = &cp
	return nil
}

// UpdateStatus 更新实例状态。
func (s *inMemoryStore) UpdateStatus(ctx context.Context, instanceID int64, status int, resultCode int, resultMsg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.m[instanceID]; ok {
		r.Status = status
		r.ResultCode = resultCode
		r.ResultMsg = resultMsg
		r.UpdatedAt = time.Now()
		return nil
	}
	return errors.New("not found")
}

// Get 按 instanceID 读取记录。
func (s *inMemoryStore) Get(ctx context.Context, instanceID int64) (*InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.m[instanceID]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, errors.New("not found")
}

// ListRunning 列出运行中实例。
func (s *inMemoryStore) ListRunning(ctx context.Context) ([]InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]InstanceRecord, 0)
	for _, v := range s.m {
		if v.Status == StateRunning {
			out = append(out, *v)
		}
	}
	return out, nil
}
