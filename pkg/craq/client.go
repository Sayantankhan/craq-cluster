package craq

import (
	"context"
	"craq-cluster/gen/rpcpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type NodeClient struct {
	conn   *grpc.ClientConn
	client rpcpb.NodeClient
}

func NewNodeClient(addr string) (*NodeClient, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &NodeClient{
		conn:   conn,
		client: rpcpb.NewNodeClient(conn),
	}, nil
}

func (c *NodeClient) Write(ctx context.Context, req *rpcpb.WriteReq) (*rpcpb.WriteAck, error) {
	return c.client.Write(ctx, req)
}

func (c *NodeClient) Read(ctx context.Context, req *rpcpb.ReadReq) (*rpcpb.ReadResponse, error) {
	return c.client.Read(ctx, req)
}

func (c *NodeClient) QueryVersion(ctx context.Context, req *rpcpb.VersionQuery) (*rpcpb.VersionResponse, error) {
	return c.client.QueryVersion(ctx, req)
}

func (c *NodeClient) Close() error {
	return c.conn.Close()
}
