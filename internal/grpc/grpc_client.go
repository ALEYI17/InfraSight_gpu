package grpc

import (
	"context"
	"fmt"

	"github.com/ALEYI17/InfraSight_gpu/internal/grpc/pb"
	"github.com/ALEYI17/InfraSight_gpu/pkg/logutil"
	"github.com/ALEYI17/InfraSight_gpu/pkg/types"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type Client struct {
	conn    *grpc.ClientConn
	client  pb.GpuEventCollectorClient
	loaders []types.Gpu_loaders
}

const maxMsgSize = 64 * 1024 * 1024

func NewGrpcClient(address string, port string, loaders []types.Gpu_loaders) (*Client, error) {
	serverAdress := fmt.Sprintf("%s:%s", address, port)
	conn, err := grpc.NewClient(serverAdress, grpc.WithTransportCredentials(insecure.NewCredentials()),
    grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(maxMsgSize), grpc.MaxCallSendMsgSize(maxMsgSize)))
	if err != nil {
		return nil, err
	}

	client := pb.NewGpuEventCollectorClient(conn)

	return &Client{conn: conn, client: client, loaders: loaders}, nil
}

func (c *Client) Close() error {
	return c.conn.Close()
}

func (c *Client) SendGpuBatch(ctx context.Context, in *pb.GpuBatch) (*pb.CollectorAck, error) {
	logger := logutil.GetLogger()

	logger.Info("Batch size", zap.Int("size", len(in.Batch)))

	return c.client.SendGpuBatch(ctx, in)

}

func (c *Client) Run(ctx context.Context, nodeName string) error {
	logger := logutil.GetLogger()

	eventCh := make(chan *pb.GpuBatch, 500)

	for _, loader := range c.loaders {

		go func(l types.Gpu_loaders) {
			tracerChannel := l.Run(ctx, nodeName)
			for {
				select {
				case <-ctx.Done():
					return
				case event := <-tracerChannel:
					eventCh <- event
				}
			}

		}(loader)
	}

	for {
		select {
		case <-ctx.Done():
			logger.Info("Client received cancellation signal")
			return nil
		case event := <-eventCh:
			_, err := c.SendGpuBatch(ctx, event)
			if err != nil {
				logger.Error("Error from sending", zap.Error(err))
				status, ok := status.FromError(err)
				if ok && (status.Code() == codes.Unavailable || status.Code() == codes.Canceled) {
					logger.Warn("Server unavailable. Shutting down client.")
					return err
				}
			}

		}
	}
}
