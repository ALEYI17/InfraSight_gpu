package grpc

import (
	"context"
	"fmt"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)


type Client struct{
  conn   *grpc.ClientConn
	client pb.GpuEventCollectorClient
  loaders []types.Gpu_loaders
}

func NewGrpcClient(address string, port string) (*Client,error){
  serverAdress := fmt.Sprintf("%s:%s", address,port)
	conn, err := grpc.NewClient(serverAdress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := pb.NewGpuEventCollectorClient(conn)

  return &Client{conn: conn,client: client},nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendBatch(ctx context.Context,in *pb.Batch) (*pb.CollectorAck, error){
  logger:= logutil.GetLogger()

  logger.Info("Batch size", zap.Int("size",len(in.Batch)))

  return c.client.SendBatch(ctx, in)

}
