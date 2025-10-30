package timeserie

import (
	"time"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
	"golang.org/x/sys/unix"
)

type GpuEventToken struct {
	Timestamp int64   `json:"timestamp"`
	EventType int64  `json:"type"`
	Value     float64 `json:"value"`
	Dir       int64  `json:"dir,omitempty"`
	PID       uint32  `json:"pid"`
	Comm      string  `json:"comm"`
}

func EventToToken(ev any) *GpuEventToken {
	now := time.Now().UnixNano()

	switch e := ev.(type) {
	case gpuprint.GpuprintGpuKernelLaunchEventT:
		return &GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_KERNEL_LAUNCH,
			Value:     float64(e.TotalThreads),
			PID:       e.Pid,
			Comm:      unix.ByteSliceToString(e.Comm[:]),
		}

	case gpuprint.GpuprintGpuMemallocEventT:
		return &GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_MALLOC,
			Value:     float64(e.ByteSize),
			PID:       e.Pid,
			Comm:      unix.ByteSliceToString(e.Comm[:]),
		}

	case gpuprint.GpuprintGpuMemcpyEventT:
		dir := types.DIR_DTOH
		if e.Kind == types.DIR_HTOD {
			dir = types.DIR_HTOD
		}
		return &GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_MEMCPY,
			Value:     float64(e.ByteSize),
			Dir:       int64(dir),
			PID:       e.Pid,
			Comm:      unix.ByteSliceToString(e.Comm[:]),
		}

	case gpuprint.GpuprintGpuStreamEventT:
		return &GpuEventToken{
			Timestamp: now,
			EventType: types.EVENT_GPU_STREAM_SYNC,
			Value:     float64(e.DeltaNs),
			PID:       e.Pid,
			Comm:      unix.ByteSliceToString(e.Comm[:]),
		}
	default:
		return nil
	}
}
