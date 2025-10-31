package timeserie

import (
	"time"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
)


func EventToToken(ev any) *pb.GpuEventToken {
	now := time.Now().UnixNano()

	switch e := ev.(type) {
	case gpuprint.GpuprintGpuKernelLaunchEventT:
		return &pb.GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_KERNEL_LAUNCH,
			Value:     float64(e.TotalThreads),
		}

	case gpuprint.GpuprintGpuMemallocEventT:
		return &pb.GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_MALLOC,
			Value:     float64(e.ByteSize),
		}

	case gpuprint.GpuprintGpuMemcpyEventT:
		dir := types.DIR_DTOH
		if e.Kind == types.DIR_HTOD {
			dir = types.DIR_HTOD
		}
		return &pb.GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_MEMCPY,
			Value:     float64(e.ByteSize),
			Dir:       int64(dir),
		}

	case gpuprint.GpuprintGpuStreamEventT:
		return &pb.GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_STREAM_SYNC,
			Value:     float64(e.DeltaNs),
		}
	default:
		return nil
	}
}
