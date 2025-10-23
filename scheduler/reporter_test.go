package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/mengeric/powerjob-client-go/client"
	"github.com/mengeric/powerjob-client-go/mocks"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

type fakeLister struct{ items []Running }

func (f fakeLister) ListRunning(ctx context.Context) ([]Running, error) { return f.items, nil }

func TestReporter(t *testing.T) {
	Convey("reporter should call ReportInstanceStatus for running instances", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		api := mocks.NewMockServerAPI(ctrl)
		// 期待对两个实例进行上报
		api.EXPECT().ReportInstanceStatus(gomock.Any(), "127.0.0.1:10010", gomock.AssignableToTypeOf(client.TaskTrackerReportInstanceStatusReq{})).Times(2)

		disc := NewDiscovery(api, 1, "127.0.0.1:10010", "0.1.0", 1)
		l := fakeLister{items: []Running{{JobID: 7, InstanceID: 1, Status: 3}, {JobID: 7, InstanceID: 2, Status: 3}}}
		rep := NewReporter(api, disc, l, "127.0.0.1:27777", 1)

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		rep.Start(ctx)
		time.Sleep(1200 * time.Millisecond)
		So(true, ShouldBeTrue)
	})
}
