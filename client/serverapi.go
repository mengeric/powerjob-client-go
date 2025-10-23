package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/mengeric/powerjob-client-go/logging"
)

// ServerAPI 定义与 PowerJob-Server 的交互接口，便于 gomock 打桩。
// 功能：封装 /server/assert、/server/acquire、心跳、状态与日志上报等。
type ServerAPI interface {
	AssertApp(ctx context.Context, bootstrapHost, appName string) (appID int64, err error)
	Acquire(ctx context.Context, base string, appID int64, currentServer, clientVersion string) (addr string, err error)
	Heartbeat(ctx context.Context, serverAddr string, hb WorkerHeartbeat) error
	ReportInstanceStatus(ctx context.Context, serverAddr string, req TaskTrackerReportInstanceStatusReq) error
	ReportLog(ctx context.Context, serverAddr string, req WorkerLogReportReq) error
}

// httpServerAPI 实现 ServerAPI。
type httpServerAPI struct{ hc *http.Client }

// NewHTTPServerAPI 构造 HTTP 实现。
func NewHTTPServerAPI() ServerAPI { return &httpServerAPI{hc: &http.Client{Timeout: 8 * time.Second}} }

// AssertApp 发起 /server/assert 校验应用是否已注册。
// 参数：bootstrapHost 形如 127.0.0.1:7700，appName 应用名。
// 返回：appID，或错误。
func (h *httpServerAPI) AssertApp(ctx context.Context, bootstrapHost, appName string) (int64, error) {
	u := fmt.Sprintf("http://%s/server/assert?appName=%s", bootstrapHost, url.QueryEscape(appName))
	var resp CommonResp[int64]
	if err := h.get(ctx, u, &resp); err != nil {
		return 0, err
	}
	if !resp.Success {
		return 0, fmt.Errorf("assert app failed: %s", resp.Message)
	}
	return resp.Data, nil
}

// Acquire 周期性获取真实调度地址。
func (h *httpServerAPI) Acquire(ctx context.Context, base string, appID int64, currentServer, clientVersion string) (string, error) {
	v := url.Values{}
	v.Set("appId", fmt.Sprintf("%d", appID))
	if currentServer != "" {
		v.Set("currentServer", currentServer)
	}
	v.Set("protocol", "HTTP")
	if clientVersion != "" {
		v.Set("clientVersion", clientVersion)
	}
	u := fmt.Sprintf("http://%s/server/acquire?%s", base, v.Encode())
	var resp CommonResp[string]
	if err := h.get(ctx, u, &resp); err != nil {
		return "", err
	}
	if !resp.Success {
		return "", fmt.Errorf("acquire failed: %s", resp.Message)
	}
	return resp.Data, nil
}

// Heartbeat 上报心跳。
func (h *httpServerAPI) Heartbeat(ctx context.Context, serverAddr string, hb WorkerHeartbeat) error {
	u := fmt.Sprintf("http://%s/server/workerHeartbeat", serverAddr)
	return h.post(ctx, u, hb, nil)
}

// ReportInstanceStatus 上报实例状态。
func (h *httpServerAPI) ReportInstanceStatus(ctx context.Context, serverAddr string, req TaskTrackerReportInstanceStatusReq) error {
	u := fmt.Sprintf("http://%s/server/reportInstanceStatus", serverAddr)
	return h.post(ctx, u, req, nil)
}

// ReportLog 上报日志。
func (h *httpServerAPI) ReportLog(ctx context.Context, serverAddr string, req WorkerLogReportReq) error {
	u := fmt.Sprintf("http://%s/server/reportLog", serverAddr)
	return h.post(ctx, u, req, nil)
}

// get 执行 GET 请求并解码 JSON。
func (h *httpServerAPI) get(ctx context.Context, url string, out any) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	res, err := h.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		b, _ := io.ReadAll(res.Body)
		return fmt.Errorf("GET %s => %d: %s", url, res.StatusCode, string(b))
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// post 执行 POST 请求并可选解码响应。
func (h *httpServerAPI) post(ctx context.Context, u string, body any, out any) error {
	b, _ := json.Marshal(body)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := h.hc.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode/100 != 2 {
		rb, _ := io.ReadAll(res.Body)
		return fmt.Errorf("POST %s => %d: %s", u, res.StatusCode, string(rb))
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}

// SafeLogErr 打印但不打断流程。
func SafeLogErr(err error, msg string) {
    if err != nil {
        logging.L().Errorf(context.Background(), "%s: %v", msg, err)
    }
}
