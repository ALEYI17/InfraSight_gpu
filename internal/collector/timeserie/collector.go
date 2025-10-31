package timeserie

import (
	"sync"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
	"golang.org/x/sys/unix"
)

type TimeSeriesCollector struct {
	mu      sync.Mutex
	buffers map[uint32][]*pb.GpuEventToken 
  comms   map[uint32]string
}

func NewTimeSeriesCollector() *TimeSeriesCollector {
	return &TimeSeriesCollector{
		buffers: make(map[uint32][]*pb.GpuEventToken),
    comms: make(map[uint32]string),
	}
}

func (tc *TimeSeriesCollector) Update(ev any) {
	token := EventToToken(ev)
	if token == nil {
		return
	}
  var pid uint32
	var comm string

	switch e := ev.(type) {
	case gpuprint.GpuprintGpuKernelLaunchEventT:
		pid, comm = e.Pid, unix.ByteSliceToString(e.Comm[:])
	case gpuprint.GpuprintGpuMemallocEventT:
		pid, comm = e.Pid, unix.ByteSliceToString(e.Comm[:])
	case gpuprint.GpuprintGpuMemcpyEventT:
		pid, comm = e.Pid, unix.ByteSliceToString(e.Comm[:])
	case gpuprint.GpuprintGpuStreamEventT:
		pid, comm = e.Pid, unix.ByteSliceToString(e.Comm[:])
	default:
		return
	}
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.buffers[pid] = append(tc.buffers[pid], token)
  tc.comms[pid] = comm
}

func (tc *TimeSeriesCollector) Flush() *pb.Batch {
	tc.mu.Lock()
	defer tc.mu.Unlock()

  var events []*pb.GpuEvent

  for pid,tokens := range tc.buffers{
    for _,tk := range tokens{
      event := &pb.GpuEvent{
      Pid: pid,
      Comm: tc.comms[pid],
      EventType: "timeserie",
        Payload: &pb.GpuEvent_Token{
          Token: tk,
        },
      }
      events = append(events, event)
    }     
  }

  batch := &pb.Batch{Batch: events,Type: "gpu_time_series"}
  return batch
}

