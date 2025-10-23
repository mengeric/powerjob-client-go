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

// 复用带锁内存存储
type memStore3 struct {
	m  map[int64]*InstanceRecord
	mu sync.RWMutex
}

func (s *memStore3) Upsert(ctx context.Context, rec *InstanceRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.m == nil {
		s.m = map[int64]*InstanceRecord{}
	}
	cp := *rec
	s.m[rec.InstanceID] = &cp
	return nil
}
func (s *memStore3) UpdateStatus(ctx context.Context, id int64, st int, code int, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.m[id]; ok {
		r.Status = st
		r.ResultCode = code
		r.ResultMsg = msg
	}
	return nil
}
func (s *memStore3) Get(ctx context.Context, id int64) (*InstanceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.m[id]; ok {
		cp := *r
		return &cp, nil
	}
	return nil, context.DeadlineExceeded
}
func (s *memStore3) ListRunning(ctx context.Context) ([]InstanceRecord, error) {
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

type dummyAPI3 struct{ client.ServerAPI }

func TestWorker_StartHTTP(t *testing.T) {
	Convey("StartHTTP should listen and handle requests on random port", t, func() {
		w := NewWorker(&memStore3{}, Options{BootstrapServer: "x", AppName: "demo", WorkerAddress: "127.0.0.1:0"}, &dummyAPI3{})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, addr, err := w.StartHTTP(ctx, "127.0.0.1:0", "/worker")
		So(err, ShouldBeNil)
		So(addr, ShouldNotEqual, "")
		time.Sleep(20 * time.Millisecond)
		// 调用 runJob
		req := client.ServerScheduleJobReq{InstanceID: 3, JobID: 7, ProcessorInfo: "simple", JobParams: `{"sleepMS": 10}`}
		b, _ := json.Marshal(req)
		resp, err := http.Post("http://"+addr+"/worker/runJob", "application/json", bytes.NewReader(b))
		So(err, ShouldBeNil)
		So(resp.StatusCode, ShouldEqual, 200)

		// 稍后查询
		time.Sleep(30 * time.Millisecond)
		qb, _ := json.Marshal(map[string]any{"instanceId": 3})
		qr, err := http.Post("http://"+addr+"/worker/queryInstanceStatus", "application/json", bytes.NewReader(qb))
		So(err, ShouldBeNil)
		So(qr.StatusCode, ShouldEqual, 200)
	})
}
