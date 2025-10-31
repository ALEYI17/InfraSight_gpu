package types

import (
	"context"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
)

type Gpu_collectors interface {
	Update(ev any)
	Flush() *pb.GpuBatch
	Run(context.Context) <-chan *pb.GpuBatch
}
