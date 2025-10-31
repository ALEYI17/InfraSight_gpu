package aggregator

import (
	"context"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
)


func (ga * GPUAggregator)Run(ctx context.Context ) <- chan *pb.Batch {
	//logger := logutil.GetLogger()
  out := make(chan *pb.Batch)

  go func(){
    defer close(out)
    ticker := time.NewTicker(ga.windowDuration)
		defer ticker.Stop()
    for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				batch := ga.Flush()
				if batch != nil {
					out <- batch
				}
			}
		}
  }()

  return out
	// for {
	// 	select {
	// 	case <-ctx.Done():
	// 		logger.Info("Context cancelled, shutting down gracefully...")
	// 		return
	//
	// 	case ev := <-events:
	// 		switch e := ev.(type) {
	// 		case gpuprint.GpuprintGpuKernelLaunchEventT:
	// 			logger.Info("GPU Event received",
	// 				zap.Uint32("pid", e.Pid),
	// 				zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
	// 				zap.Uint32("blockx", e.Blockx),
	// 				zap.Uint32("blocky", e.Blocky),
	// 				zap.Uint32("blockz", e.Blockz),
	// 				zap.Uint32("gridx", e.Gridx),
	// 				zap.Uint32("gridy", e.Gridy),
	// 				zap.Uint32("gridz", e.Gridz),
	// 				zap.Uint64("threadsblock", e.ThreadsBlock),
	// 				zap.Uint64("totalblocks", e.TotalBlocks),
	// 				zap.Any("threads", e.TotalThreads))
	// 		case gpuprint.GpuprintGpuMemallocEventT:
	// 			logger.Info("Gpu event received",
	// 				zap.Uint32("pid", e.Pid),
	// 				zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
	// 				zap.Uint64("Bytes", e.ByteSize))
	// 		case gpuprint.GpuprintGpuMemcpyEventT:
	// 			logger.Info("Gpu event received",
	// 				zap.Uint32("pid", e.Pid),
	// 				zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
	// 				zap.Uint64("Bytes", e.ByteSize),
	// 				zap.Uint8("kind", e.Kind))
	// 		case gpuprint.GpuprintGpuStreamEventT:
	// 			logger.Info("Gpu event received",
	// 				zap.Uint32("pid", e.Pid),
	// 				zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
	// 				zap.Uint64("delta", e.DeltaNs))
	// 		}
	// 		agg.Update(ev)
	//
	// 	case <-ticker.C:
	// 		summaries := agg.Flush()
	// 		for _, s := range summaries {
	// 			logger.Info("Aggregated GPU fingerprint",
	// 				zap.Uint32("pid", s.PID),
	// 				zap.String("comm", s.Comm),
	// 				zap.Float64("launch_rate", s.LaunchRate),
	// 				zap.Float64("memcpy_rate", s.MemcpyRate),
	// 				zap.Float64("alloc_rate", s.AllocRate),
	// 				zap.Float64("avg_threads_kernel", s.AvgThreadsPerKernel),
	// 				zap.Uint64("total_threads", s.TotalThreadsLaunched),
	// 				zap.Uint64("total_memcpy_bytes", s.TotalMemcpyBytes),
	// 				zap.Float64("htod_ratio", s.HTODRatio),
	// 				zap.Float64("avg_sync_ns", s.AvgSyncTimeNs),
	// 				zap.Float64("sync_fraction", s.SyncFraction),
	// 			)
	// 		}
	// 	}
	// }
}
