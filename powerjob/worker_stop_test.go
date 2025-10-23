package powerjob

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/mengeric/powerjob-client-go/client"
	. "github.com/smartystreets/goconvey/convey"
)

// 复用带锁的内存存储
type memStore2 struct {
	m  map[int64]*InstanceRecord
	mu sync.RWMutex
}

func (s *memStore2) Upsert(ctx context.Context, rec *InstanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[int64]*InstanceRecord{}
	}
	cp := *rec
	s.m[rec.InstanceID] = &cp
	return nil
}
func (s *memStore2) UpdateStatus(ctx context.Context, id int64, st int, code int, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.m[id]; ok {
		r.Status = st
		r.ResultCode = code
		r.ResultMsg = msg
	}
	return nil
}
func (s *memStore2) Get(ctx context.Context, id int64) (*InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.m[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, context.DeadlineExceeded
}
func (s *memStore2) ListRunning(ctx context.Context) ([]InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []InstanceRecord
	for _, v := range s.m {
		if v.Status == StateRunning {
			out = append(out, *v)
		}
	}
	return out, nil
}

type dummyAPI2 struct{ client.ServerAPI }

func (d *dummyAPI2) AssertApp(ctx context.Context, host, app string) (int64, error) { return 1, nil }

func TestWorker_StopInstance(t *testing.T) {
	Convey("stopInstance should cancel running job", t, func() {
		w := NewWorker(withStore(&memStore2{}), WithBootstrapServer("x"), WithAppName("demo"), WithListenAddr("127.0.0.1:0"), WithClientAPI(&dummyAPI2{}))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go w.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		addr := w.Addr()

		// 启动一个较长的任务
		req := client.ServerScheduleJobReq{InstanceID: 2, JobID: 7, ProcessorInfo: "simple", JobParams: `{"sleepMS": 500}`}
		b, _ := json.Marshal(req)
		_, _ = http.Post("http://"+addr+"/worker/runJob", "application/json", bytes.NewReader(b))

		// 立即下发停止
		sb, _ := json.Marshal(map[string]any{"instanceId": 2})
		_, _ = http.Post("http://"+addr+"/worker/stopInstance", "application/json", bytes.NewReader(sb))

		// 等待一小会，查询应非 Running
		time.Sleep(120 * time.Millisecond)
		qb, _ := json.Marshal(map[string]any{"instanceId": 2})
		qr, err := http.Post("http://"+addr+"/worker/queryInstanceStatus", "application/json", bytes.NewReader(qb))
		So(err, ShouldBeNil)
		So(qr.StatusCode, ShouldEqual, 200)
	})
}
