package types

import (
	"context"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
)

type Gpu_loaders interface {
	Close()
	Run(context.Context, string) <-chan *pb.GpuBatch
}
