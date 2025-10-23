package powerjob

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"net"
	"os"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/logging"
	"github.com/mengeric/powerjob-client-go/processor"
	"github.com/mengeric/powerjob-client-go/scheduler"
	"github.com/mengeric/powerjob-client-go/tracker"
)

// Worker 组件主对象：对外提供 HTTP Handler 与生命周期控制。
// 注意：Worker 不创建 HTTP Server；宿主负责将 Handler 挂载到自身路由。
type Worker struct {
	opt   Options
	api   client.ServerAPI
	store Storage

	trk  *tracker.Manager
	disc *scheduler.Discovery
    hb   *scheduler.HeartbeatScheduler
    rep  *scheduler.InstanceReporter
    lr   *scheduler.LogReporter
}

// NewWorker 创建 Worker。
// 参数：
// - store：实例持久化实现，必填；
// - opt：运行参数，若周期为零将使用默认值；
// - api：与 Server 通讯客户端，nil 时使用默认 HTTP 实现。
// 返回：*Worker 与错误（当前构造阶段不返回错误）。
func NewWorker(store Storage, opt Options, api client.ServerAPI) *Worker {
	opt.withDefaults()
	if api == nil {
		api = client.NewHTTPServerAPI()
	}
	return &Worker{opt: opt, api: api, store: store, trk: tracker.NewManager()}
}

// Start 启动后台调度（服务发现/心跳/实例上报）。
// 异常：网络失败不抛出，内部日志记录并重试。
func (w *Worker) Start(ctx context.Context) {
	// App 校验与获取 appId
	appID, err := w.api.AssertApp(ctx, w.opt.BootstrapServer, w.opt.AppName)
	if err != nil {
		logging.L().Warn(ctx, "assert app failed", "err", err)
	}

	// Discovery/Heartbeat/Reporter
	w.disc = scheduler.NewDiscovery(w.api, appID, w.opt.BootstrapServer, w.opt.ClientVersion, int(w.opt.DiscoveryEvery.Seconds()))
	w.disc.Start(ctx)

	w.hb = scheduler.NewHeartbeat(w.api, w.disc, w.opt.WorkerAddress, int(w.opt.HeartbeatEvery.Seconds()))
	w.hb.Start(ctx)

    w.rep = scheduler.NewReporter(w.api, w.disc, listerAdapter{w.store}, w.opt.WorkerAddress, int(w.opt.ReportEvery.Seconds()))
    w.rep.Start(ctx)

    w.lr = scheduler.NewLogReporter(w.api, w.disc, w.opt.WorkerAddress, int(w.opt.LogReportEvery.Seconds()), w.opt.LogBatchSize)
    w.lr.Start(ctx)
}

// MountHTTP 将组件的 HTTP 路由挂载到宿主 mux，base 前缀默认为 /worker。
// 端点：POST {base}/runJob、{base}/stopInstance、{base}/queryInstanceStatus
func (w *Worker) MountHTTP(mux *http.ServeMux, base string) {
	if base == "" {
		base = "/worker"
	}
	mux.HandleFunc(base+"/runJob", w.handleRunJob)
	mux.HandleFunc(base+"/stopInstance", w.handleStopInstance)
	mux.HandleFunc(base+"/queryInstanceStatus", w.handleQueryInstanceStatus)
}

// StartHTTP 创建并启动一个内置的 HTTP Server，监听指定地址并挂载组件路由。
// 功能：快速把组件跑起来，宿主无需自行创建 http.Server。
// 参数：
//   - ctx: 控制生命周期，ctx.Done() 时将优雅关闭；
//   - addr: 监听地址，支持形如 ":27777"、"127.0.0.1:0"（0 表示随机端口）；
//   - base: 路由前缀，留空默认 "/worker"。
//
// 返回：
//   - srv: *http.Server 实例，便于观察状态；
//   - actualAddr: 实际监听地址（当传入 :0 时用于获取随机端口）；
//   - err: 启动失败时返回错误。
func (w *Worker) StartHTTP(ctx context.Context, addr, base string) (srv *http.Server, actualAddr string, err error) {
	if base == "" {
		base = "/worker"
	}
	mux := http.NewServeMux()
	w.MountHTTP(mux, base)

	// 处理 :0 随机端口以便拿到实际端口
	ln, lerr := net.Listen("tcp", addr)
	if lerr != nil {
		return nil, "", lerr
	}
	actualAddr = ln.Addr().String()

	srv = &http.Server{Addr: actualAddr, Handler: mux}
	go func() {
		// 当 ctx 结束时优雅关闭
		<-ctx.Done()
		_ = srv.Shutdown(context.Background())
	}()
	go func() { _ = srv.Serve(ln) }()
	return srv, actualAddr, nil
}

// StartHTTPWithSignals 创建并启动 HTTP Server（携带系统信号监听）。
// 功能：等价于 WithSignalCancel(context.Background()) + StartHTTP(...) 的组合，便于快速接入优雅关闭。
// 参数：
//   - addr：监听地址；
//   - base：路由前缀；
//   - sigs：可选信号列表，留空默认 SIGINT、SIGTERM。
//
// 返回：
//   - srv：*http.Server；
//   - actualAddr：实际监听地址；
//   - stop：释放 signal 监听的函数（通常 defer stop()）；
//   - err：错误。
func (w *Worker) StartHTTPWithSignals(addr, base string, sigs ...os.Signal) (srv *http.Server, actualAddr string, stop context.CancelFunc, err error) {
	ctx, cancel := WithSignalCancel(context.Background(), sigs...)
	s, a, e := w.StartHTTP(ctx, addr, base)
	if e != nil {
		cancel()
		return nil, "", nil, e
	}
	return s, a, cancel, nil
}

// handleRunJob 任务执行入口（Server -> Worker）。
func (w *Worker) handleRunJob(rw http.ResponseWriter, r *http.Request) {
	var req client.ServerScheduleJobReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(rw, http.StatusBadRequest, err)
		return
	}
	if _, ok := w.trk.Get(req.InstanceID); ok {
		rw.WriteHeader(http.StatusOK)
		return
	}
	_ = w.store.Upsert(r.Context(), &InstanceRecord{InstanceID: req.InstanceID, JobID: req.JobID, Status: StateRunning, StartedAt: time.Now(), UpdatedAt: time.Now()})
	ins := w.trk.Start(req.InstanceID)
	go w.execute(r.Context(), req, ins)
	rw.WriteHeader(http.StatusOK)
}

// execute 实例执行与状态更新。
func (w *Worker) execute(ctx context.Context, req client.ServerScheduleJobReq, ins *tracker.Instance) {
	p, ok := processor.Get(req.ProcessorInfo)
	if !ok {
		_ = w.store.UpdateStatus(context.Background(), req.InstanceID, StateFailed, -1, "processor not found")
		w.trk.Stop(req.InstanceID)
		return
	}
    // 直接把原始 JSON 字节传给处理器，由处理器自行解码
    res, err := p.Run(ins.Ctx, []byte(req.JobParams))
	if err != nil {
		_ = w.store.UpdateStatus(context.Background(), req.InstanceID, StateFailed, res.Code, err.Error())
	} else {
		_ = w.store.UpdateStatus(context.Background(), req.InstanceID, StateSucceed, res.Code, res.Msg)
	}
	w.trk.Stop(req.InstanceID)
}

// Log 推送一条在线日志（供处理器或业务调用）。
// level: 1=DEBUG, 2=INFO, 3=WARN, 4=ERROR；timeMs：日志时间（毫秒）。
func (w *Worker) Log(instanceID int64, level int, content string, timeMs int64) {
    if w.lr == nil { return }
    if timeMs == 0 { timeMs = time.Now().UnixMilli() }
    w.lr.Enqueue(client.InstanceLogContent{InstanceID: instanceID, LogContent: content, LogLevel: level, LogTime: timeMs})
}

// handleStopInstance 停止实例执行。
func (w *Worker) handleStopInstance(rw http.ResponseWriter, r *http.Request) {
	var body struct {
		InstanceID int64 `json:"instanceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(rw, http.StatusBadRequest, err)
		return
	}
	if w.trk.Stop(body.InstanceID) {
		_ = w.store.UpdateStatus(r.Context(), body.InstanceID, StateStopped, 0, "stopped")
	}
	rw.WriteHeader(http.StatusOK)
}

// handleQueryInstanceStatus 查询实例状态。
func (w *Worker) handleQueryInstanceStatus(rw http.ResponseWriter, r *http.Request) {
	var body struct {
		InstanceID int64 `json:"instanceId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(rw, http.StatusBadRequest, err)
		return
	}
	rec, err := w.store.Get(r.Context(), body.InstanceID)
	if err != nil {
		writeErr(rw, http.StatusNotFound, err)
		return
	}
	writeJSON(rw, rec)
}

// writeErr/JSON 公共返回工具。
func writeErr(w http.ResponseWriter, code int, err error) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "message": err.Error()})
}
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// storageAdapter 适配调度器对 repo 的依赖（仅用到 ListRunning）。
type listerAdapter struct{ Storage }

// ListRunning 将组件存储模型映射为调度器精简视图。
func (a listerAdapter) ListRunning(ctx context.Context) ([]scheduler.Running, error) {
	recs, err := a.Storage.ListRunning(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]scheduler.Running, 0, len(recs))
	for _, r := range recs {
		out = append(out, scheduler.Running{JobID: r.JobID, InstanceID: r.InstanceID, Status: r.Status})
	}
	return out, nil
}
