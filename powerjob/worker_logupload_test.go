package powerjob

import (
    "bytes"
    "context"
    "encoding/json"
    "net/http"
    "sync/atomic"
    "testing"
    "time"

    "github.com/mengeric/powerjob-client-go/client"
    "github.com/mengeric/powerjob-client-go/logging"
    "github.com/mengeric/powerjob-client-go/processor"
    . "github.com/smartystreets/goconvey/convey"
)

// test processor：在 Run 中使用 logging.L() 写出一条带实例上下文的日志
type logProc struct{}
func (p *logProc) GetTaskKey() string { return "logproc" }
func (p *logProc) Init(ctx context.Context) error { return nil }
func (p *logProc) Stop(ctx context.Context) error { return nil }
func (p *logProc) Run(ctx context.Context, raw []byte) (processor.Result, error) {
    // 使用日志门面；Hook 会把日志写入在线日志通道
    logging.L().Infof(ctx, "hello k=%d", 1)
    return processor.Result{Code:0, Msg:"ok"}, nil
}

// mock ServerAPI：捕获 ReportLog 调用次数
type logAPI struct{ client.ServerAPI; count int32 }
func (l *logAPI) AssertApp(ctx context.Context, host, app string) (int64, error) { return 1, nil }
func (l *logAPI) ReportLog(ctx context.Context, addr string, req client.WorkerLogReportReq) error {
    atomic.AddInt32(&l.count, int32(len(req.InstanceLogContents)))
    return nil
}

func TestWorker_LogUpload_Hook(t *testing.T) {
    Convey("component logs with instance context should be uploaded", t, func() {
        processor.Register(&logProc{})
        api := &logAPI{}
        w := NewWorker(
            WithBootstrapServer("x"),
            WithAppName("demo"),
            WithListenAddr("127.0.0.1:0"),
            WithClientAPI(api),
            // 上报周期 1 秒
            WithLogReporter(1*time.Second, 16),
        )
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        go w.Start(ctx)
        time.Sleep(60 * time.Millisecond)
        addr := w.Addr()

        // 触发一次执行；虽然 logProc 未显式写日志，但组件本身在执行路径上会产生一些日志
        req := client.ServerScheduleJobReq{InstanceID: 55, JobID: 1, ProcessorInfo: "logproc", JobParams: `{}`}
        b, _ := json.Marshal(req)
        _, _ = http.Post("http://"+addr+"/worker/runJob", "application/json", bytes.NewReader(b))

        // 等待一轮 flush
        time.Sleep(1200 * time.Millisecond)
        So(atomic.LoadInt32(&api.count), ShouldBeGreaterThan, 0)
    })
}
