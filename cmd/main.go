package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"go.uber.org/zap"
)

func main(){
  _, cancel := context.WithCancel(context.Background())
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
}
