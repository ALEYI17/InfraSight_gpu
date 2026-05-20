package loaders

import (
	"context"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"go.uber.org/zap"
)

const (
    WarmupSeconds    = 3
    IoctlMinThreshold = 50
    PollInterval     = 5 * time.Second
)

type WatchdogAlert struct {
    Pid  uint32
    Comm string
}

type pidSnapshot struct {
    ioctlCount  uint64
    uprobeCount uint64
    firstSeenNs uint64
}

type WatchdogCollector struct {
    objs      *gpuprint.GpuprintObjects
    snapshots map[uint32]pidSnapshot  // pid → last snapshot
}

func NewWatchdogCollector(objs *gpuprint.GpuprintObjects) *WatchdogCollector {
    return &WatchdogCollector{
        objs:      objs,
        snapshots: make(map[uint32]pidSnapshot),
    }
}

func (w *WatchdogCollector) Run(ctx context.Context) <-chan WatchdogAlert {
    alerts := make(chan WatchdogAlert, 10)
    logger := logutil.GetLogger()

    go func() {
        defer close(alerts)
        ticker := time.NewTicker(PollInterval)
        defer ticker.Stop()

        for {
            select {
            case <-ctx.Done():
                return
            case <-ticker.C:
                w.poll(alerts, logger)
            }
        }
    }()

    return alerts
}

func (w *WatchdogCollector) poll(alerts chan<- WatchdogAlert, logger *zap.Logger) {
    nowNs := uint64(time.Now().UnixNano())

    // iterate all entries in the PERCPU map
    var pid uint32
    var perCpuValues []gpuprint.GpuprintIoctlWatchdogEventT

    iter := w.objs.IoctlWatchdogMap.Iterate()
    for iter.Next(&pid, &perCpuValues) {

        // sum across CPUs (PERCPU map)
        var totalIoctl  uint64
        var totalUprobe uint64
        var firstSeen   uint64

        for _, v := range perCpuValues {
            totalIoctl  += v.IoctlHitCount
            totalUprobe += v.UprobeHitCount
            // first seen: take the minimum non-zero across CPUs
            if v.FirstSeenTime > 0 {
                if firstSeen == 0 || v.FirstSeenTime < firstSeen {
                    firstSeen = v.FirstSeenTime
                }
            }
        }

        // skip if below minimum activity
        if totalIoctl < IoctlMinThreshold {
            continue
        }

        // warmup check — convert ns to seconds
        ageNs := nowNs - firstSeen
        if ageNs < uint64(WarmupSeconds)*1_000_000_000 {
            continue
        }

        // get previous snapshot
        prev, seen := w.snapshots[pid]

        if !seen {
            // first time seeing this pid — just store snapshot
            w.snapshots[pid] = pidSnapshot{
                ioctlCount:  totalIoctl,
                uprobeCount: totalUprobe,
                firstSeenNs: firstSeen,
            }
            continue
        }

        // compute deltas since last poll
        deltaIoctl  := totalIoctl  - prev.ioctlCount
        deltaUprobe := totalUprobe - prev.uprobeCount

        // update snapshot
        w.snapshots[pid] = pidSnapshot{
            ioctlCount:  totalIoctl,
            uprobeCount: totalUprobe,
            firstSeenNs: firstSeen,
        }

        // skip idle windows
        if deltaIoctl < IoctlMinThreshold {
            continue
        }

        // THE detection condition
        if deltaUprobe == 0 {
            logger.Warn("WATCHDOG: GPU activity bypassing system libcuda",
                zap.Uint32("pid", pid),
                zap.Uint64("delta_ioctl", deltaIoctl),
            )
            alerts <- WatchdogAlert{Pid: pid}
        }
    }

    if err := iter.Err(); err != nil {
        logger.Error("watchdog map iteration error", zap.Error(err))
    }
}
