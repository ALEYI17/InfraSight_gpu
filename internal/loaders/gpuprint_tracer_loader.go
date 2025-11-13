package loaders

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"go.uber.org/zap"
)

type GpuprintLoader struct {
	Objs       *gpuprint.GpuprintObjects
	Up         []link.Link
	Rb         *ringbuf.Reader
	collectors []types.Gpu_collectors
}

func NewGpuprinterLoader(collectors ...types.Gpu_collectors) (*GpuprintLoader, error) {

	logger := logutil.GetLogger()
	if err := rlimit.RemoveMemlock(); err != nil {
		return nil, err
	}

	objs := gpuprint.GpuprintObjects{}

	if err := gpuprint.LoadGpuprintObjects(&objs, nil); err != nil {
		logger.Error("error", zap.Error(err))
		return nil, err
	}

	gput := &GpuprintLoader{
		Objs: &objs,
	}

	ex, err := link.OpenExecutable("/usr/lib/wsl/drivers/nvmdi.inf_amd64_c2d1126d336032b3/libcuda.so.1.1")
	if err != nil {
		logger.Error("error", zap.Error(err))
		gput.Close()
		return nil, err
	}

	functions_entry := []struct {
		name string
		prog *ebpf.Program
	}{
    // kernel launch
		{"cuLaunchKernel", objs.HandleCuLaunchkernel},
    {"cuLaunchKernel_ptsz", objs.HandleCuLaunchkernel},
    // CuMemAlloc
		{"cuMemAlloc_v2", objs.HandleCuMemAlloc},
    {"cuMemAlloc", objs.HandleCuMemAlloc},
    // cuMemcpyHtoD
		{"cuMemcpyHtoD_v2", objs.HandleCuMemcpyHtod},
		{"cuMemcpyHtoDAsync_v2", objs.HandleCuMemcpyHtodAsync},
    {"cuMemcpyHtoD", objs.HandleCuMemcpyHtod},
		{"cuMemcpyHtoDAsync", objs.HandleCuMemcpyHtodAsync},
    {"cuMemcpyHtoDAsync_v2_ptsz", objs.HandleCuMemcpyHtod},
		{"cuMemcpyHtoD_v2_ptds", objs.HandleCuMemcpyHtodAsync},
    // cuMemDtoH
		{"cuMemcpyDtoH_v2", objs.HandleCuMemcpyDtoh},
		{"cuMemcpyDtoHAsync_v2", objs.HandleCuMemcpyDtohAsync},
    {"cuMemcpyDtoH", objs.HandleCuMemcpyDtoh},
		{"cuMemcpyDtoHAsync", objs.HandleCuMemcpyDtohAsync},
    {"cuMemcpyDtoHAsync_v2_ptsz", objs.HandleCuMemcpyDtoh},
		{"cuMemcpyDtoH_v2_ptds", objs.HandleCuMemcpyDtohAsync},
    // sync stream
		{"cuStreamSynchronize", objs.HandleCuStreamSync},
    {"cuStreamSynchronize_ptsz", objs.HandleCuStreamSync},
    // Sync ctx
    {"cuCtxSynchronize", objs.HandleCuCtxSync},
    {"cuCtxSynchronize_v2", objs.HandleCuCtxSync},
	}

	functions_exit := []struct {
		name string
		prog *ebpf.Program
	}{
		{"cuStreamSynchronize", objs.HandleCuStreamSynchronizeRet},
    {"cuStreamSynchronize_ptsz", objs.HandleCuStreamSynchronizeRet},
    {"cuCtxSynchronize", objs.HandleCuCtxSyncRet},
    {"cuCtxSynchronize_v2", objs.HandleCuCtxSyncRet},
	}

	for _, fn := range functions_entry {
		up, err := ex.Uprobe(fn.name, fn.prog, nil)
		if err != nil {
			logger.Warn("failed to attach uprobe", zap.String("function", fn.name), zap.Error(err))
			continue // skip this one but keep others
		}
		gput.Up = append(gput.Up, up)
		logger.Info("attached uprobe", zap.String("function", fn.name))
	}

	for _, fn := range functions_exit {
		up, err := ex.Uretprobe(fn.name, fn.prog, nil)
		if err != nil {
			logger.Warn("failed to attach uretprobe", zap.String("function", fn.name), zap.Error(err))
			continue // skip this one but keep others
		}
		gput.Up = append(gput.Up, up)
		logger.Info("attached uretprobe", zap.String("function", fn.name))
	}

	rb, err := ringbuf.NewReader(objs.GpuRingbuf)
	if err != nil {
		logger.Error("error", zap.Error(err))
		gput.Close()
		return nil, err
	}

	gput.Rb = rb

	for _, c := range collectors {
		gput.collectors = append(gput.collectors, c)
	}

	return gput, nil
}

func (gt *GpuprintLoader) Close() {
	if gt.Objs != nil {
		gt.Objs.Close()
	}
	for _, up := range gt.Up {
		if up != nil {
			up.Close()
		}
	}
	if gt.Rb != nil {
		gt.Rb.Close()
	}
}

func (gt *GpuprintLoader) Run(ctx context.Context, nodeName string) <-chan *pb.GpuBatch {

	out := make(chan *pb.GpuBatch)

	logger := logutil.GetLogger()

	for _, c := range gt.collectors {
		go func(col types.Gpu_collectors) {
			for batch := range col.Run(ctx) {
				out <- batch
			}
		}(c)
	}

	go func() {

		for {
			select {
			case <-ctx.Done():
				logger.Info("Context cancelled, stopping loader...")
				return
			default:
				record, err := gt.Rb.Read()
				if err != nil {
					if errors.Is(err, ringbuf.ErrClosed) {
						logger.Info("Ring buffer closed, exiting...")
						return
					}
					logger.Error("Reading error", zap.Error(err))
					continue
				}

				if len(record.RawSample) < 1 {
					logger.Warn("Empty record")
					continue
				}

				flag := record.RawSample[0]

				switch flag {
				case types.EVENT_GPU_KERNEL_LAUNCH: // EVENT_GPU_KERNEL_LAUNCH
					var e gpuprint.GpuprintGpuKernelLaunchEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing kernel launch event", zap.Error(err))
						continue
					}
					gt.sendToCollectors(e)

				case types.EVENT_GPU_MALLOC: // EVENT_GPU_MALLOC
					var e gpuprint.GpuprintGpuMemallocEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing memalloc event", zap.Error(err))
						continue
					}
					gt.sendToCollectors(e)

				case types.EVENT_GPU_MEMCPY: // EVENT_GPU_MEMCPY
					var e gpuprint.GpuprintGpuMemcpyEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing memcpy event", zap.Error(err))
						continue
					}
					gt.sendToCollectors(e)
				case types.EVENT_GPU_STREAM_SYNC: // EVENT_GPU_STREAM_SYNC
					var e gpuprint.GpuprintGpuStreamEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing stream sync event", zap.Error(err))
						continue
					}
					gt.sendToCollectors(e)

				default:
					logger.Warn("Unknown event flag", zap.Uint8("flag", flag))
					continue
				}
			}
		}

	}()

	return out
}

func (gt *GpuprintLoader) sendToCollectors(e any) {
	for _, c := range gt.collectors {
		c.Update(e)
	}
}
