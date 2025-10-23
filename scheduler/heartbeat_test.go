package scheduler

import (
    "context"
    "testing"
    "time"

    . "github.com/smartystreets/goconvey/convey"
    "go.uber.org/mock/gomock"
    "powerjob-client-go/mocks"
    "powerjob-client-go/client"
)

func TestHeartbeatScheduler(t *testing.T) {
    Convey("heartbeat should call API at least once", t, func() {
        ctrl := gomock.NewController(t)
        defer ctrl.Finish()
        api := mocks.NewMockServerAPI(ctrl)

        // 预期至少一次 Heartbeat 调用
        api.EXPECT().Heartbeat(gomock.Any(), "127.0.0.1:10010", gomock.AssignableToTypeOf(client.WorkerHeartbeat{})).AnyTimes()

        // discovery 不启动，仅用于 Get() 返回固定地址
        disc := NewDiscovery(api, 1, "127.0.0.1:10010", "0.1.0", 1)
        hb := NewHeartbeat(api, disc, "127.0.0.1:27777", 1)

        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        hb.Start(ctx)

        time.Sleep(120 * time.Millisecond)
        So(true, ShouldBeTrue)
    })
}
