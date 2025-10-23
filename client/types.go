package client

// 以下类型根据多语言文档抽象，字段命名与说明文档一致或等价。

// CommonResp 统一响应包装。
type CommonResp[T any] struct {
	Success bool   `json:"success"`
	Data    T      `json:"data"`
	Message string `json:"message"`
}

// ServerScheduleJobReq runJob 请求体（Server -> Worker）。
type ServerScheduleJobReq struct {
	AllWorkerAddress   []string `json:"allWorkerAddress"`
	JobID              int64    `json:"jobId"`
	WfInstanceID       *int64   `json:"wfInstanceId"`
	InstanceID         int64    `json:"instanceId"`
	ExecuteType        string   `json:"executeType"`
	ProcessorType      string   `json:"processorType"`
	ProcessorInfo      string   `json:"processorInfo"`
	InstanceTimeout    int64    `json:"instanceTimeoutMS"`
	JobParams          string   `json:"jobParams"`
	InstanceParams     *string  `json:"instanceParams"`
	ThreadConcurrency  int      `json:"threadConcurrency"`
	TaskRetryNum       int      `json:"taskRetryNum"`
	TimeExpressionType string   `json:"timeExpressionType"`
	TimeExpression     *string  `json:"timeExpression"`
	MaxInstanceNum     int      `json:"maxInstanceNum"`
}

// WorkerHeartbeat 心跳包。
type WorkerHeartbeat struct {
	WorkerAddress string       `json:"workerAddress"`
	HeartbeatTime int64        `json:"heartbeatTime"`
	Protocol      string       `json:"protocol"` // 固定 HTTP
	SystemMetrics SystemMetric `json:"systemMetrics"`
}

// SystemMetric 系统指标。
type SystemMetric struct {
	CPULoad        float64 `json:"cpuLoad"`
	CPUProcessors  int     `json:"cpuProcessors"`
	DiskTotalGB    float64 `json:"diskTotal"`
	DiskUsageRatio float64 `json:"diskUsage"`
	DiskUsedGB     float64 `json:"diskUsed"`
	ProcMaxMemory  float64 `json:"jvmMaxMemory"`
	ProcMemUsage   float64 `json:"jvmMemoryUsage"`
	ProcUsedMemory float64 `json:"jvmUsedMemory"`
	Score          float64 `json:"score"`
}

// TaskTrackerReportInstanceStatusReq 实例状态上报。
type TaskTrackerReportInstanceStatusReq struct {
	JobID          int64  `json:"jobId"`
	InstanceID     int64  `json:"instanceId"`
	ReportTime     int64  `json:"reportTime"`
	SourceAddress  string `json:"sourceAddress"`
	InstanceStatus int    `json:"instanceStatus"`
}

// WorkerLogReportReq 在线日志上报。
type WorkerLogReportReq struct {
	InstanceLogContents []InstanceLogContent `json:"instanceLogContents"`
	WorkerAddress       string               `json:"workerAddress"`
}

// InstanceLogContent 单条日志。
type InstanceLogContent struct {
	InstanceID int64  `json:"instanceId"`
	LogContent string `json:"logContent"`
	LogLevel   int    `json:"logLevel"` // 1~4: DEBUG/INFO/WARN/ERROR
	LogTime    int64  `json:"logTime"`
}
