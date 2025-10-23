package scheduler

import (
	"context"
	"testing"
	"time"

	"github.com/mengeric/powerjob-client-go/mocks"
	. "github.com/smartystreets/goconvey/convey"
	"go.uber.org/mock/gomock"
)

func TestDiscovery(t *testing.T) {
	Convey("discovery should refresh address", t, func() {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()
		api := mocks.NewMockServerAPI(ctrl)
		api.EXPECT().Acquire(gomock.Any(), "boot:7700", int64(1), "boot:7700", "0.1.0").Return("10.0.0.1:10010", nil).AnyTimes()
		d := NewDiscovery(api, 1, "boot:7700", "0.1.0", 1)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		d.Start(ctx)
		time.Sleep(1200 * time.Millisecond)
		So(d.Get(), ShouldEqual, "10.0.0.1:10010")
	})
}
