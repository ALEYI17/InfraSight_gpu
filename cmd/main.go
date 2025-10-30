package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/internal/collector/aggregator"
	"github.com/ALEYI17/InfraSight_gpu/internal/loaders"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"go.uber.org/zap"
)

func main() {
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

	gl, err := loaders.NewGpuprinterLoader()
	if err != nil {
		logger.Fatal("err", zap.Error(err))
	}
	defer gl.Close()

	
  aggregator.RunWithAggregation(ctx, gl, 10*time.Second)
}
