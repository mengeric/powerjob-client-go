package powerjob

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "sync"
    "time"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/logging"
	"github.com/mengeric/powerjob-client-go/processor"
	"github.com/mengeric/powerjob-client-go/scheduler"
	"github.com/mengeric/powerjob-client-go/tracker"
)

// Worker 组件主对象：提供内置 HTTP Server 与后台调度生命周期控制。
// 说明：Worker 在 Start(ctx) 中自动启动 HTTP Server（监听 Options.ListenAddr），
// 并开启服务发现、心跳、实例状态与在线日志上报任务。
type Worker struct {
    opt   Options
    api   client.ServerAPI
    store Storage

	trk    *tracker.Manager
	disc   *scheduler.Discovery
	hb     *scheduler.HeartbeatScheduler
	rep    *scheduler.InstanceReporter
	lr     *scheduler.LogReporter
	srv    *http.Server
	addrMu sync.RWMutex
	addr   string
}

// NewWorker 创建 Worker。
// 功能：按照 With... 可选项组合出一个可运行的 Worker；若未显式传入存储实现，默认使用内置内存存储。
// 参数：
// - opts：一组 Option，可配置引导地址、应用名、监听端口、心跳与上报周期、日志上报策略等；
// 返回：
// - *Worker：已初始化的 Worker 实例；
// 异常：
// - 构造阶段不返回错误，运行时问题会在 Start 时通过日志输出并重试。
func NewWorker(opts ...Option) *Worker {
	// 默认：内存存储 + HTTP ServerAPI
	cfg := &workerConfig{opt: Options{}, store: nil, api: nil}
	for _, fn := range opts {
		fn(cfg)
	}
	cfg.opt.withDefaults()
    w := &Worker{opt: cfg.opt, trk: tracker.NewManager()}
	if cfg.store != nil {
		w.store = cfg.store
	} else {
		// 避免 import cycle：默认使用包内置的内存实现
		w.store = newDefaultMemStore()
	}
	if cfg.api != nil {
		if a, ok := cfg.api.(client.ServerAPI); ok {
			w.api = a
		}
	}
	if w.api == nil {
		w.api = client.NewHTTPServerAPI()
	}
	return w
}

// Start 启动后台调度（服务发现/心跳/实例上报）。
// 功能：
// 1) 先启动内置 HTTP Server 并确定对外地址（可能为随机端口），必要时回填 WorkerAddress；
// 2) 执行应用断言获取 appId；
// 3) 启动服务发现、心跳、实例状态与在线日志上报任务；
// 生命周期：受传入 ctx 控制，ctx.Done 时优雅关闭 HTTP Server 并停止后台协程。
// 异常：网络失败不抛出，内部日志记录并按周期重试。
func (w *Worker) Start(ctx context.Context) {
	// 1) 内置 HTTP Server：先启动监听并确定实际地址
	mux := http.NewServeMux()
	w.registerHandlers(mux, "/worker")
	ln, err := net.Listen("tcp", w.opt.ListenAddr)
    if err != nil {
        logging.L().Errorf(ctx, "listen failed: addr=%s err=%v", w.opt.ListenAddr, err)
        return
    }
	w.addrMu.Lock()
	w.addr = ln.Addr().String()
	w.addrMu.Unlock()
	if w.opt.WorkerAddress == "" {
		w.opt.WorkerAddress = w.addr
	}
	w.srv = &http.Server{Addr: w.addr, Handler: mux}
	go func() { <-ctx.Done(); _ = w.srv.Shutdown(context.Background()) }()
	go func() { _ = w.srv.Serve(ln) }()

	// 2) App 校验与获取 appId
	appID, err := w.api.AssertApp(ctx, w.opt.BootstrapServer, w.opt.AppName)
    if err != nil {
        logging.L().Warnf(ctx, "assert app failed: %v", err)
    }

	// 3) Discovery/Heartbeat/Reporter/LogReporter
	w.disc = scheduler.NewDiscovery(w.api, appID, w.opt.BootstrapServer, w.opt.ClientVersion, int(w.opt.DiscoveryEvery.Seconds()))
	w.disc.Start(ctx)

	w.hb = scheduler.NewHeartbeat(w.api, w.disc, w.opt.WorkerAddress, int(w.opt.HeartbeatEvery.Seconds()))
	w.hb.Start(ctx)

    w.rep = scheduler.NewReporter(w.api, w.disc, listerAdapter{Storage: w.store}, w.opt.WorkerAddress, int(w.opt.ReportEvery.Seconds()))
	w.rep.Start(ctx)

    w.lr = scheduler.NewLogReporter(w.api, w.disc, w.opt.WorkerAddress, int(w.opt.LogReportEvery.Seconds()), w.opt.LogBatchSize)
    w.lr.Start(ctx)
    // 设置日志上传 Hook：当上下文携带实例ID时，自动将日志通过在线日志通道上报
    logging.SetHook(w.uploadHook)
}

// MountHTTP 将组件的 HTTP 路由挂载到宿主 mux，base 前缀默认为 /worker。
// 端点：POST {base}/runJob、{base}/stopInstance、{base}/queryInstanceStatus
func (w *Worker) registerHandlers(mux *http.ServeMux, base string) {
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
// 去除对外暴露的 StartHTTP/MountHTTP：组件在 Start(ctx) 内部自动启动 HTTP 服务

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
    // 将实例ID注入上下文，便于日志 Hook 识别并在线上报
    ins.Ctx = withInstanceID(ins.Ctx, req.InstanceID)
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
	if w.lr == nil {
		return
	}
	if timeMs == 0 {
		timeMs = time.Now().UnixMilli()
	}
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

// Addr 返回内置 HTTP Server 的实际监听地址（用于测试或 :0 随机端口场景）。
func (w *Worker) Addr() string { w.addrMu.RLock(); defer w.addrMu.RUnlock(); return w.addr }

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

// ---- 日志上传 Hook 与实例上下文工具 ----

// ctxKey 用于在 Context 中存放实例ID，避免与外部键冲突。
type ctxKey string

var ctxKeyIID ctxKey = "powerjob_iid"

// withInstanceID 将实例ID写入 Context。
// 参数：ctx 原始上下文；id 实例ID。
// 返回：包含实例ID的新上下文。
func withInstanceID(ctx context.Context, id int64) context.Context { return context.WithValue(ctx, ctxKeyIID, id) }

// instanceIDFromContext 尝试从上下文中提取实例ID。
func instanceIDFromContext(ctx context.Context) (int64, bool) {
    v := ctx.Value(ctxKeyIID)
    if v == nil { return 0, false }
    if id, ok := v.(int64); ok { return id, true }
    return 0, false
}

// uploadHook 将带实例上下文的日志写入在线日志队列。
// level：1=DEBUG,2=INFO,3=WARN,4=ERROR。
// 注意：Hook 不得再次调用 logging.L()，以避免递归。
func (w *Worker) uploadHook(ctx context.Context, level int, msg string, args ...any) {
    if w.lr == nil { return }
    iid, ok := instanceIDFromContext(ctx)
    if !ok || iid == 0 { return }
    // 组装内容：msg | k=v ...
    content := msg
    // 简单扁平化 key-value
    if len(args) > 0 {
        content += " |"
        for i := 0; i < len(args); i++ {
            if i%2 == 0 {
                // 键
                if k, ok := args[i].(string); ok {
                    content += " " + k + "="
                } else {
                    content += " arg="
                }
            } else {
                content += toString(args[i])
            }
        }
    }
    w.Log(iid, level, content, 0)
}

// toString 将任意值转为字符串，避免引入 fmt 分配热点。
func toString(v any) string {
    switch x := v.(type) {
    case string:
        return x
    case []byte:
        return string(x)
    case int:
        return itoa(int64(x))
    case int64:
        return itoa(x)
    case uint64:
        return itoa(int64(x))
    case bool:
        if x { return "true" } else { return "false" }
    default:
        // 回退到标准库格式化
        return fmtAny(v)
    }
}

// itoa 简化版整型转字符串。
func itoa(x int64) string {
    b := [20]byte{}
    i := len(b)
    neg := x < 0
    if neg { x = -x }
    for x >= 10 {
        i--
        q := x / 10
        b[i] = byte('0' + x - q*10)
        x = q
    }
    i--
    b[i] = byte('0' + x)
    if neg {
        i--
        b[i] = '-'
    }
    return string(b[i:])
}

// fmtAny 使用 fmt.Sprint，将其封装以便未来替换。
func fmtAny(v any) string { return fmt.Sprint(v) }
