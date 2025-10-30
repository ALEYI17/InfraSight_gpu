package aggregator

import (
	"sync"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
	"golang.org/x/sys/unix"
)

type GPUAggregator struct {
	windows        map[uint32]*GPUFingerprint
	mu             sync.Mutex
	windowDuration time.Duration
	lastFlush      time.Time
}

func NewGPUAggregator(window time.Duration) *GPUAggregator {
	return &GPUAggregator{
		windows:        make(map[uint32]*GPUFingerprint),
		windowDuration: window,
		lastFlush:      time.Now(),
	}
}

func (ga *GPUAggregator) ensureWindow(pid uint32, comm []byte) *GPUFingerprint {
	win, ok := ga.windows[pid]
	if !ok {
		now := time.Now()
		win = &GPUFingerprint{
			PID:         pid,
			Comm:        unix.ByteSliceToString(comm[:]),
			WindowStart: now,
			WindowEnd:   now.Add(ga.windowDuration),
		}
		ga.windows[pid] = win
	}
	return win
}

func max(a, b uint64) uint64 {
	if a > b {
		return a
	}
	return b
}

func (ga *GPUAggregator) Update(ev any) {
	ga.mu.Lock()
	defer ga.mu.Unlock()

	switch e := ev.(type) {
	case gpuprint.GpuprintGpuKernelLaunchEventT:
		w := ga.ensureWindow(e.Pid, e.Comm[:])
		w.KernelLaunchCount++
		totalThreads := e.TotalThreads
		w.TotalThreadsLaunched += totalThreads
		w.AvgThreadsPerKernel = ((w.AvgThreadsPerKernel * float64(w.KernelLaunchCount-1)) + float64(totalThreads)) / float64(w.KernelLaunchCount)
		w.MaxThreadsPerKernel = max(w.MaxThreadsPerKernel, totalThreads)

	case gpuprint.GpuprintGpuMemallocEventT:
		w := ga.ensureWindow(e.Pid, e.Comm[:])
		w.MemAllocCount++
		w.TotalMemAllocBytes += e.ByteSize

	case gpuprint.GpuprintGpuMemcpyEventT:
		w := ga.ensureWindow(e.Pid, e.Comm[:])
		w.MemcpyCount++
		w.TotalMemcpyBytes += e.ByteSize
		if e.Kind == types.DIR_HTOD {
			w.HTODBytes += e.ByteSize
		} else {
			w.DTOHBytes += e.ByteSize
		}

	case gpuprint.GpuprintGpuStreamEventT:
		w := ga.ensureWindow(e.Pid, e.Comm[:])
		w.StreamSyncCount++
		w.AvgSyncTimeNs = ((w.AvgSyncTimeNs * float64(w.StreamSyncCount-1)) + float64(e.DeltaNs)) / float64(w.StreamSyncCount)
		w.MaxSyncTimeNs = max(w.MaxSyncTimeNs, e.DeltaNs)
	}
}

func (ga *GPUAggregator) Flush() []GPUFingerprint {
	ga.mu.Lock()
	defer ga.mu.Unlock()

	now := time.Now()
	finished := []GPUFingerprint{}

	for pid, w := range ga.windows {
		if now.After(w.WindowEnd) {
			duration := w.WindowEnd.Sub(w.WindowStart).Seconds()
			if w.TotalMemcpyBytes > 0 {
				w.HTODRatio = float64(w.HTODBytes) / float64(w.TotalMemcpyBytes)
			}
			totalEvents := w.KernelLaunchCount + w.MemcpyCount + w.MemAllocCount + w.StreamSyncCount
			if totalEvents > 0 {
				w.SyncFraction = float64(w.StreamSyncCount) / float64(totalEvents)
			}
			if duration > 0 {
				w.LaunchRate = float64(w.KernelLaunchCount) / duration
				w.MemcpyRate = float64(w.MemcpyCount) / duration
				w.AllocRate = float64(w.MemAllocCount) / duration
			}

			finished = append(finished, *w)
			delete(ga.windows, pid)
		}
	}

	ga.lastFlush = now
	return finished
}
