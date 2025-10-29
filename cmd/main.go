package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ALEYI17/InfraSight_gpu/bpf/cuda/gpuprint"
	"github.com/ALEYI17/InfraSight_gpu/internal/loaders"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"
)

func main(){
  ctx, cancel := context.WithCancel(context.Background())
  logutil.InitLogger()

  logger := logutil.GetLogger()
  defer logger.Sync()

	go func() {
		sigch := make(chan os.Signal, 1)
		signal.Notify(sigch, syscall.SIGINT, syscall.SIGTERM)
		sig := <-sigch
    logger.Info("Received signal, shutting down", zap.String("signal", sig.String()))
		cancel()
	}()

  gl,err := loaders.NewGpuprinterLoader()
  if err !=nil{
    logger.Fatal("err", zap.Error(err))
  }
  defer gl.Close()

  events:= gl.Run(ctx, "Casa")

  for {
    select {
    case <-ctx.Done():
        logger.Info("Context cancelled, shutting down gracefully...")
        return
    case ev := <-events:
        logger.Info("get event")
        switch e := ev.(type){
          case gpuprint.GpuprintGpuKernelLaunchEventT:
            logger.Info("GPU Event received",
              zap.Uint32("pid", e.Pid),
          zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
              zap.Uint32("blockx", e.Blockx),
              zap.Uint32("blocky", e.Blocky),
              zap.Uint32("blockz", e.Blockz),
              zap.Uint32("gridx", e.Gridx),
              zap.Uint32("gridy", e.Gridy),
              zap.Uint32("gridz", e.Gridz),
              zap.Uint64("threadsblock",e.ThreadsBlock),
              zap.Uint64("totalblocks",e.TotalBlocks),
              zap.Any("threads", e.TotalThreads))  
          case gpuprint.GpuprintGpuMemallocEventT:
            logger.Info("Gpu event received", 
              zap.Uint32("pid", e.Pid),
              zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
              zap.Uint64("Bytes", e.ByteSize))
          case gpuprint.GpuprintGpuMemcpyEventT:
            logger.Info("Gpu event received", 
              zap.Uint32("pid", e.Pid),
              zap.String("comm", unix.ByteSliceToString(e.Comm[:])),
              zap.Uint64("Bytes", e.ByteSize),
              zap.Uint8("kind", e.Kind))
        }
        
    }
  }

}
