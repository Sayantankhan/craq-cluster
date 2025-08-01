// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.5.1
// - protoc             v5.29.3
// source: node.proto

package rpcpb

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.64.0 or later.
const _ = grpc.SupportPackageIsVersion9

const (
	Node_StreamWrite_FullMethodName  = "/rpcpb.Node/StreamWrite"
	Node_StreamRead_FullMethodName   = "/rpcpb.Node/StreamRead"
	Node_QueryVersion_FullMethodName = "/rpcpb.Node/QueryVersion"
	Node_ListFiles_FullMethodName    = "/rpcpb.Node/ListFiles"
)

// NodeClient is the client API for Node service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
//
// gRPC service for CRAQ nodes
type NodeClient interface {
	// stream
	StreamWrite(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[StreamWriteReq, WriteAck], error)
	StreamRead(ctx context.Context, in *StreamReadReq, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadChunk], error)
	QueryVersion(ctx context.Context, in *VersionQuery, opts ...grpc.CallOption) (*VersionResponse, error)
	// List all files in a folder
	ListFiles(ctx context.Context, in *FolderQuery, opts ...grpc.CallOption) (*FileList, error)
}

type nodeClient struct {
	cc grpc.ClientConnInterface
}

func NewNodeClient(cc grpc.ClientConnInterface) NodeClient {
	return &nodeClient{cc}
}

func (c *nodeClient) StreamWrite(ctx context.Context, opts ...grpc.CallOption) (grpc.ClientStreamingClient[StreamWriteReq, WriteAck], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Node_ServiceDesc.Streams[0], Node_StreamWrite_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StreamWriteReq, WriteAck]{ClientStream: stream}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Node_StreamWriteClient = grpc.ClientStreamingClient[StreamWriteReq, WriteAck]

func (c *nodeClient) StreamRead(ctx context.Context, in *StreamReadReq, opts ...grpc.CallOption) (grpc.ServerStreamingClient[ReadChunk], error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	stream, err := c.cc.NewStream(ctx, &Node_ServiceDesc.Streams[1], Node_StreamRead_FullMethodName, cOpts...)
	if err != nil {
		return nil, err
	}
	x := &grpc.GenericClientStream[StreamReadReq, ReadChunk]{ClientStream: stream}
	if err := x.ClientStream.SendMsg(in); err != nil {
		return nil, err
	}
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	return x, nil
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Node_StreamReadClient = grpc.ServerStreamingClient[ReadChunk]

func (c *nodeClient) QueryVersion(ctx context.Context, in *VersionQuery, opts ...grpc.CallOption) (*VersionResponse, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(VersionResponse)
	err := c.cc.Invoke(ctx, Node_QueryVersion_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *nodeClient) ListFiles(ctx context.Context, in *FolderQuery, opts ...grpc.CallOption) (*FileList, error) {
	cOpts := append([]grpc.CallOption{grpc.StaticMethod()}, opts...)
	out := new(FileList)
	err := c.cc.Invoke(ctx, Node_ListFiles_FullMethodName, in, out, cOpts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// NodeServer is the server API for Node service.
// All implementations must embed UnimplementedNodeServer
// for forward compatibility.
//
// gRPC service for CRAQ nodes
type NodeServer interface {
	// stream
	StreamWrite(grpc.ClientStreamingServer[StreamWriteReq, WriteAck]) error
	StreamRead(*StreamReadReq, grpc.ServerStreamingServer[ReadChunk]) error
	QueryVersion(context.Context, *VersionQuery) (*VersionResponse, error)
	// List all files in a folder
	ListFiles(context.Context, *FolderQuery) (*FileList, error)
	mustEmbedUnimplementedNodeServer()
}

// UnimplementedNodeServer must be embedded to have
// forward compatible implementations.
//
// NOTE: this should be embedded by value instead of pointer to avoid a nil
// pointer dereference when methods are called.
type UnimplementedNodeServer struct{}

func (UnimplementedNodeServer) StreamWrite(grpc.ClientStreamingServer[StreamWriteReq, WriteAck]) error {
	return status.Errorf(codes.Unimplemented, "method StreamWrite not implemented")
}
func (UnimplementedNodeServer) StreamRead(*StreamReadReq, grpc.ServerStreamingServer[ReadChunk]) error {
	return status.Errorf(codes.Unimplemented, "method StreamRead not implemented")
}
func (UnimplementedNodeServer) QueryVersion(context.Context, *VersionQuery) (*VersionResponse, error) {
	return nil, status.Errorf(codes.Unimplemented, "method QueryVersion not implemented")
}
func (UnimplementedNodeServer) ListFiles(context.Context, *FolderQuery) (*FileList, error) {
	return nil, status.Errorf(codes.Unimplemented, "method ListFiles not implemented")
}
func (UnimplementedNodeServer) mustEmbedUnimplementedNodeServer() {}
func (UnimplementedNodeServer) testEmbeddedByValue()              {}

// UnsafeNodeServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to NodeServer will
// result in compilation errors.
type UnsafeNodeServer interface {
	mustEmbedUnimplementedNodeServer()
}

func RegisterNodeServer(s grpc.ServiceRegistrar, srv NodeServer) {
	// If the following call pancis, it indicates UnimplementedNodeServer was
	// embedded by pointer and is nil.  This will cause panics if an
	// unimplemented method is ever invoked, so we test this at initialization
	// time to prevent it from happening at runtime later due to I/O.
	if t, ok := srv.(interface{ testEmbeddedByValue() }); ok {
		t.testEmbeddedByValue()
	}
	s.RegisterService(&Node_ServiceDesc, srv)
}

func _Node_StreamWrite_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(NodeServer).StreamWrite(&grpc.GenericServerStream[StreamWriteReq, WriteAck]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Node_StreamWriteServer = grpc.ClientStreamingServer[StreamWriteReq, WriteAck]

func _Node_StreamRead_Handler(srv interface{}, stream grpc.ServerStream) error {
	m := new(StreamReadReq)
	if err := stream.RecvMsg(m); err != nil {
		return err
	}
	return srv.(NodeServer).StreamRead(m, &grpc.GenericServerStream[StreamReadReq, ReadChunk]{ServerStream: stream})
}

// This type alias is provided for backwards compatibility with existing code that references the prior non-generic stream type by name.
type Node_StreamReadServer = grpc.ServerStreamingServer[ReadChunk]

func _Node_QueryVersion_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(VersionQuery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).QueryVersion(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Node_QueryVersion_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).QueryVersion(ctx, req.(*VersionQuery))
	}
	return interceptor(ctx, in, info, handler)
}

func _Node_ListFiles_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(FolderQuery)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(NodeServer).ListFiles(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: Node_ListFiles_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(NodeServer).ListFiles(ctx, req.(*FolderQuery))
	}
	return interceptor(ctx, in, info, handler)
}

// Node_ServiceDesc is the grpc.ServiceDesc for Node service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Node_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "rpcpb.Node",
	HandlerType: (*NodeServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "QueryVersion",
			Handler:    _Node_QueryVersion_Handler,
		},
		{
			MethodName: "ListFiles",
			Handler:    _Node_ListFiles_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "StreamWrite",
			Handler:       _Node_StreamWrite_Handler,
			ClientStreams: true,
		},
		{
			StreamName:    "StreamRead",
			Handler:       _Node_StreamRead_Handler,
			ServerStreams: true,
		},
	},
	Metadata: "node.proto",
}
