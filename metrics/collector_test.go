package metrics

import (
	"context"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func TestCollectSystemMetric(t *testing.T) {
	Convey("collect metrics should not panic and be in range", t, func() {
		ctx := context.Background()
		m := CollectSystemMetric(ctx)
		So(m.CPUProcessors, ShouldBeGreaterThanOrEqualTo, 1)
		So(m.Score, ShouldBeGreaterThanOrEqualTo, 0)
		So(m.Score, ShouldBeLessThanOrEqualTo, 100)
	})
}
