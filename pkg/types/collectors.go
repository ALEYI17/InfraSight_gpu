package types

import (
	"context"
	"time"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
)

type Gpu_collectors interface{
  Update(ev any)
  Flush() *pb.Batch
  Run(context.Context,time.Duration) <- chan *pb.Batch
}
