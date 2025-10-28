package loaders

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/ringbuf"
	"github.com/cilium/ebpf/rlimit"
	"go.uber.org/zap"
)

type GpuprintLoader struct{
  Objs *gpuprint.GpuprintObjects
  Up link.Link
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

  ex,err := link.OpenExecutable("/usr/local/cuda-13.0/targets/x86_64-linux/lib/libcudart.so")
  if err != nil {
    logger.Error("error", zap.Error(err))
    gput.Close()
    return nil, err
  }

  up,err := ex.Uprobe("cudaLaunchKernel", objs.HandleCudaLaunchkernel , nil)
  if err != nil {
    logger.Error("error", zap.Error(err))
    gput.Close()
    return nil, err
  }

  gput.Up = up

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
  if gt.Up !=nil {
    gt.Up.Close()
  }
  if gt.Rb !=nil {
    gt.Rb.Close()
  }
}

func (gt *GpuprintLoader) Run(ctx context.Context, nodeName string) <-chan *gpuprint.GpuprintGpuEventT{
  
  var events gpuprint.GpuprintGpuEventT

  c:= make(chan *gpuprint.GpuprintGpuEventT)

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
        if err := binary.Read(bytes.NewBuffer(record.RawSample), binary.LittleEndian, &events); err != nil {
					logger.Error("Parsing ringbuffer events", zap.Error(err))
					continue
				}
        
        select{
        case <- ctx.Done():
          logger.Info("Context cancelled while sending event...")
					return
        case c<- &events:
        }
      }
    }

  }()

  return c
}
