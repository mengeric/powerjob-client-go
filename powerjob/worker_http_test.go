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

// memStore 简易内存实现（加锁），仅用于测试，避免竞态。
type memStore struct {
	m  map[int64]*InstanceRecord
	mu sync.RWMutex
}

func (s *memStore) Upsert(ctx context.Context, rec *InstanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[int64]*InstanceRecord{}
	}
	cp := *rec
	s.m[rec.InstanceID] = &cp
	return nil
}
func (s *memStore) UpdateStatus(ctx context.Context, id int64, st int, code int, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.m[id]; ok {
		r.Status = st
		r.ResultCode = code
		r.ResultMsg = msg
	}
	return nil
}
func (s *memStore) Get(ctx context.Context, id int64) (*InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.m[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, context.DeadlineExceeded
}
func (s *memStore) ListRunning(ctx context.Context) ([]InstanceRecord, error) {
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

// dummyAPI 仅为使 NewWorker 构造成功，测试不发起外部请求。
type dummyAPI struct{ client.ServerAPI }

func (d *dummyAPI) AssertApp(ctx context.Context, host, app string) (int64, error) { return 1, nil }

func TestWorkerHTTP_RunJobFlow(t *testing.T) {
	Convey("worker runJob -> succeed", t, func() {
		w := NewWorker(WithBootstrapServer("x"), WithAppName("demo"), WithListenAddr("127.0.0.1:0"), WithClientAPI(&dummyAPI{}))
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go w.Start(ctx)
		time.Sleep(50 * time.Millisecond)
		addr := w.Addr()

		req := client.ServerScheduleJobReq{InstanceID: 1, JobID: 7, ProcessorInfo: "simple", JobParams: `{"sleepMS": 10}`}
		b, _ := json.Marshal(req)
		resp, err := http.Post("http://"+addr+"/worker/runJob", "application/json", bytes.NewReader(b))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, 200)

		// 等待执行结束
		time.Sleep(40 * time.Millisecond)
		// 查询状态
		qb, _ := json.Marshal(map[string]any{"instanceId": 1})
		qr, err := http.Post("http://"+addr+"/worker/queryInstanceStatus", "application/json", bytes.NewReader(qb))
		So(err, ShouldBeNil)
		So(qr.StatusCode, ShouldEqual, 200)
	})
}
