package tracker

import (
    "context"
    "sync"
)

// State 常量由 powerjob 包导出；这里保持独立只做运行跟踪

// Instance 维护实例运行中的上下文与取消句柄。
type Instance struct {
    Ctx    context.Context
    Cancel context.CancelFunc
    Status int
}

// Manager 简单的实例跟踪器。
type Manager struct {
    mu       sync.RWMutex
    running  map[int64]*Instance
}

// NewManager 构造。
func NewManager() *Manager { return &Manager{running: map[int64]*Instance{}} }

// Start 注册实例。
func (m *Manager) Start(id int64) *Instance {
    m.mu.Lock(); defer m.mu.Unlock()
    ctx, cancel := context.WithCancel(context.Background())
    ins := &Instance{Ctx: ctx, Cancel: cancel}
    m.running[id] = ins
    return ins
}

// Stop 取消实例。
func (m *Manager) Stop(id int64) bool {
    m.mu.Lock(); defer m.mu.Unlock()
    if ins, ok := m.running[id]; ok {
        ins.Cancel()
        delete(m.running, id)
        return true
    }
    return false
}

// Get 查询实例。
func (m *Manager) Get(id int64) (*Instance, bool) {
    m.mu.RLock(); defer m.mu.RUnlock()
    ins, ok := m.running[id]
    return ins, ok
}

// ListIDs 返回当前运行实例ID集合。
func (m *Manager) ListIDs() []int64 {
    m.mu.RLock(); defer m.mu.RUnlock()
    ids := make([]int64, 0, len(m.running))
    for id := range m.running { ids = append(ids, id) }
    return ids
}

