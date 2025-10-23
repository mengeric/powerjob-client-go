package powerjob

import (
    "context"
    "os"
    "os/signal"
    "syscall"
)

// WithSignalCancel 创建一个可响应系统信号（如 SIGINT/SIGTERM）的上下文。
// 功能：在收到进程关闭信号时自动取消返回的 Context，用于触发优雅关闭（如 HTTP Server Shutdown）。
// 参数：
//  - parent：父级上下文；
//  - signals：可选信号列表，留空则默认使用 SIGINT、SIGTERM。
// 返回：
//  - ctx：当接收到任一信号时 Done() 即会关闭；
//  - stop：释放底层 signal 监听的函数，通常在退出时 defer 调用。
// 异常：无异常；调用者自行处理 ctx 取消后的清理逻辑。
func WithSignalCancel(parent context.Context, signals ...os.Signal) (context.Context, context.CancelFunc) {
    if len(signals) == 0 {
        signals = []os.Signal{syscall.SIGINT, syscall.SIGTERM}
    }
    // Go 1.16+ 提供 signal.NotifyContext，内部会创建并返回可取消的 Context
    return signal.NotifyContext(parent, signals...)
}

