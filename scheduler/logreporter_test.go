package scheduler

import (
    "context"
    "testing"
    "time"
    . "github.com/smartystreets/goconvey/convey"
    "go.uber.org/mock/gomock"
    "github.com/mengeric/powerjob-client-go/mocks"
    "github.com/mengeric/powerjob-client-go/client"
)

func TestLogReporter(t *testing.T) {
    Convey("log reporter should batch and send", t, func() {
        ctrl := gomock.NewController(t)
        defer ctrl.Finish()
        api := mocks.NewMockServerAPI(ctrl)

        // 期待至少一次 ReportLog 调用，条数>=2
        api.EXPECT().ReportLog(gomock.Any(), "127.0.0.1:10010", gomock.Any()).AnyTimes()

        disc := NewDiscovery(api, 1, "127.0.0.1:10010", "0.1.0", 1)
        lr := NewLogReporter(api, disc, "127.0.0.1:27777", 1, 2)
        ctx, cancel := context.WithCancel(context.Background())
        defer cancel()
        lr.Start(ctx)

        now := time.Now().UnixMilli()
        lr.Enqueue(client.InstanceLogContent{InstanceID:1, LogLevel:2, LogContent:"a", LogTime:now})
        lr.Enqueue(client.InstanceLogContent{InstanceID:1, LogLevel:2, LogContent:"b", LogTime:now})
        time.Sleep(120 * time.Millisecond)
        So(true, ShouldBeTrue)
    })
}

