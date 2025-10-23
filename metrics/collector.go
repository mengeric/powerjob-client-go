package metrics

import (
	"context"
	"os"
	"runtime"
	"time"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/shirou/gopsutil/v3/disk"
	"github.com/shirou/gopsutil/v3/load"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/process"
)

// CollectSystemMetric 采集系统/进程指标并映射为心跳上报结构。
func CollectSystemMetric(ctx context.Context) client.SystemMetric {
	var out client.SystemMetric
	if avg, err := load.AvgWithContext(ctx); err == nil {
		out.CPULoad = avg.Load1
	}
	out.CPUProcessors = runtime.NumCPU()
	if du, err := disk.UsageWithContext(ctx, "/"); err == nil && du.Total > 0 {
		out.DiskTotalGB = float64(du.Total) / (1024 * 1024 * 1024)
		out.DiskUsedGB = float64(du.Used) / (1024 * 1024 * 1024)
		out.DiskUsageRatio = du.UsedPercent / 100.0
	}
	if vm, err := mem.VirtualMemoryWithContext(ctx); err == nil && vm.Total > 0 {
		out.ProcMaxMemory = float64(vm.Total) / (1024 * 1024 * 1024)
	}
	if p, err := process.NewProcess(int32(os.Getpid())); err == nil {
		if pm, err := p.MemoryInfoWithContext(ctx); err == nil && pm != nil {
			usedGB := float64(pm.RSS) / (1024 * 1024 * 1024)
			out.ProcUsedMemory = usedGB
			if out.ProcMaxMemory > 0 {
				out.ProcMemUsage = usedGB / out.ProcMaxMemory
			}
		}
	}
	score := 100.0
	if out.CPULoad > 0 {
		score -= out.CPULoad * 5
	}
	if out.DiskUsageRatio > 0 {
		score -= out.DiskUsageRatio * 20
	}
	if out.ProcMemUsage > 0 {
		score -= out.ProcMemUsage * 30
	}
	if score < 0 {
		score = 0
	}
	out.Score = score
	return out
}

func SleepForSampling() { time.Sleep(50 * time.Millisecond) }
