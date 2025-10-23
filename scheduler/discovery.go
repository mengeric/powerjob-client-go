package scheduler

import (
	"context"
	"sync/atomic"
	"time"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/logging"
)

// Discovery 周期性刷新真实 server 地址。
type Discovery struct {
	api       client.ServerAPI
	appID     int64
	bootstrap string
	version   string
	interval  time.Duration
	running   atomic.Bool
	current   atomic.Value // string
}

// NewDiscovery 构造实例。
func NewDiscovery(api client.ServerAPI, appID int64, bootstrap, version string, seconds int) *Discovery {
	d := &Discovery{api: api, appID: appID, bootstrap: bootstrap, version: version, interval: time.Duration(seconds) * time.Second}
	d.current.Store(bootstrap)
	return d
}

// Start 启动定时任务。
func (d *Discovery) Start(ctx context.Context) {
	if d.running.Swap(true) {
		return
	}
	ticker := time.NewTicker(d.interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cur := d.Get()
				addr, err := d.api.Acquire(ctx, d.bootstrap, d.appID, cur, d.version)
                if err != nil {
                    logging.L().Warnf(ctx, "acquire server failed: %v", err)
                    continue
                }
				if addr != "" {
					d.current.Store(addr)
				}
			}
		}
	}()
}

// Get 返回当前 server 地址。
func (d *Discovery) Get() string {
	v := d.current.Load()
	if v == nil {
		return d.bootstrap
	}
	return v.(string)
}
