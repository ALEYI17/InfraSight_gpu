package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/ALEYI17/InfraSight_gpu/internal/config"
	"github.com/ALEYI17/InfraSight_gpu/internal/grpc"
	"github.com/ALEYI17/InfraSight_gpu/internal/loaders"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
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

	cfg := config.LoadConfig()

	var lds []types.Gpu_loaders

	for _, program := range cfg.EnableProbes {
		if loaderInstance, err := loaders.NewEbpfGpuLoaders(program); err == nil {
			logger.Info("Loaded tracer:", zap.String("Loader", program))
			defer loaderInstance.Close()
			lds = append(lds, loaderInstance)
      logger.Info("Load successfully loader:", zap.String("Loader", program))
			continue
		} else {
			logger.Error("error to load tracer", zap.String("program", program), zap.Error(err))
		}
		
	}

	logger.Info("Loader(s) created successfully")

	client, err := grpc.NewGrpcClient(cfg.ServerAdress, cfg.Serverport, lds)
	if err != nil {
		logger.Fatal("Error creating the client", zap.Error(err))
	}

	logger.Info(" gRPC Client created successfully")

	defer client.Close()

	if err := client.Run(ctx, cfg.Nodename); err != nil {
		logger.Error("Error running client", zap.Error(err))
		return
	}
	logger.Info("Client finished running")
}
