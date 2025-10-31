package timeserie

import (
	"context"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
)

func (tc *TimeSeriesCollector) Run(ctx context.Context) <-chan *pb.GpuBatch {

	out := make(chan *pb.GpuBatch)

	go func() {
		defer close(out)
		ticker := time.NewTicker(tc.flushInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				batch := tc.Flush()
				if batch != nil {
					out <- batch
				}
			}
		}
	}()

	return out
}
