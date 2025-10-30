package aggregator

import "time"

type GPUFingerprint struct {
	PID         uint32
	Comm        string
	WindowStart time.Time
	WindowEnd   time.Time

	// Counts
	KernelLaunchCount uint64
	MemAllocCount     uint64
	MemcpyCount       uint64
	StreamSyncCount   uint64

	// Kernel metrics
	AvgThreadsPerKernel  float64
	MaxThreadsPerKernel  uint64
	AvgBlocksPerKernel   float64
	TotalThreadsLaunched uint64

	// Memory metrics
	TotalMemAllocBytes uint64
	AvgMemAllocBytes   float64
	TotalMemcpyBytes   uint64
	AvgMemcpyBytes     float64
	HTODBytes          uint64
	DTOHBytes          uint64
	HTODRatio          float64

	// Synchronization
	AvgSyncTimeNs float64
	MaxSyncTimeNs uint64
	SyncFraction  float64

	// Derived ratios
	LaunchRate float64
	MemcpyRate float64
	AllocRate  float64
}
