package loaders

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"go.uber.org/zap"
)

type GpuprintLoader struct{
  Objs *gpuprint.GpuprintObjects
  Up []link.Link
  Rb *ringbuf.Reader
}

func NewGpuprinterLoader() (*GpuprintLoader, error){

  logger := logutil.GetLogger()
  if err := rlimit.RemoveMemlock(); err != nil {
		return nil, err
	}

  objs := gpuprint.GpuprintObjects{}

  if err := gpuprint.LoadGpuprintObjects(&objs, nil); err !=nil{
    logger.Error("error", zap.Error(err))
    return nil, err
  }

  gput := &GpuprintLoader{
    Objs: &objs,
  }

  ex,err := link.OpenExecutable("/usr/lib/wsl/drivers/nvmdi.inf_amd64_f088ae99b5a2f5fd/libcuda.so.1.1")
  if err != nil {
    logger.Error("error", zap.Error(err))
    gput.Close()
    return nil, err
  }

  functions_entry := []struct {
      name string
      prog *ebpf.Program
  }{
      {"cuLaunchKernel", objs.HandleCuLaunchkernel},
      {"cuMemAlloc_v2", objs.HandleCuMemAlloc},
      {"cuMemcpyHtoD_v2", objs.HandleCuMemcpyHtod},
      {"cuMemcpyHtoDAsync_v2", objs.HandleCuMemcpyHtodAsync},
      {"cuMemcpyDtoH_v2", objs.HandleCuMemcpyDtoh},
      {"cuMemcpyDtoHAsync_v2", objs.HandleCuMemcpyDtohAsync},
      {"cuStreamSynchronize", objs.HandleCuStreamSync},
  }

  functions_exit := []struct {
      name string
      prog *ebpf.Program
  }{
      {"cuStreamSynchronize", objs.HandleCuStreamSynchronizeRet},
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

  return gput,nil
}

func (gt *GpuprintLoader) Close(){
  if gt.Objs !=nil{
    gt.Objs.Close()
  }
  for _, up := range gt.Up {
      if up != nil {
          up.Close()
      }
  }
  if gt.Rb !=nil {
    gt.Rb.Close()
  }
}

func (gt *GpuprintLoader) Run(ctx context.Context, nodeName string) <-chan any{

  c:= make(chan any)

  logger := logutil.GetLogger()

  go func (){

    for{
      select{
      case <-ctx.Done():
        logger.Info("Context cancelled, stopping loader...")
				return
      default:
        record,err := gt.Rb.Read()
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
        fmt.Printf("Raw sample (%d bytes):", len(record.RawSample))
				for i := 0; i < len(record.RawSample) && i < 32; i++ {
					fmt.Printf(" %02x", record.RawSample[i])
				}
				fmt.Println()

				flag := record.RawSample[0]

        switch flag {
				case 1: // EVENT_GPU_KERNEL_LAUNCH
					var e gpuprint.GpuprintGpuKernelLaunchEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing kernel launch event", zap.Error(err))
						continue
					}
					c <- e

				case 2: // EVENT_GPU_MALLOC
					var e gpuprint.GpuprintGpuMemallocEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing memalloc event", zap.Error(err))
						continue
					}
					c <- e

				case 3: // EVENT_GPU_MEMCPY
					var e gpuprint.GpuprintGpuMemcpyEventT
					if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err != nil {
						logger.Error("Parsing memcpy event", zap.Error(err))
						continue
					}
					c <- e
        case 4: // EVENT_GPU_STREAM_SYNC
          var e gpuprint.GpuprintGpuStreamEventT
          if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &e); err !=nil{
            logger.Error("Parsing stream sync event", zap.Error(err))
            continue
          }
          c <- e

				default:
					logger.Warn("Unknown event flag", zap.Uint8("flag", flag))
					continue
				}
      }
    }

  }()

  return c
}
