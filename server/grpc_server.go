package server

import (
    "context"
    "github.com/d3code/zlog"
    "google.golang.org/grpc"
    "google.golang.org/grpc/reflection"
    "net"
)

type GrpcServer struct {
    Host             string
    Port             string
    RegisterServices func(server *grpc.Server)
}

func (s *GrpcServer) Address() string {
    return s.Host + ":" + s.Port
}

func (s *GrpcServer) Run() {
    listen, err := net.Listen("tcp", s.Address())
    if err != nil {
        zlog.Log.Errorf("Failed to listen: %s", err)
        return
    }

    x := grpc.UnaryInterceptor(serverInterceptor)

    // Create gRPC server
    server := grpc.NewServer(x)
    reflection.Register(server)

    // Register services
    s.RegisterServices(server)

    if listen == nil || listen.Addr() == nil || server == nil {
        zlog.Log.Errorf("Failed to listen or create server")
        return
    }

    // Start gRPC server
    zlog.Log.Infof("Starting gRPC server on %s", listen.Addr().String())
    err = server.Serve(listen)
    if err != nil {
        zlog.Log.Errorf("Failed to serve: %s", err)
    }
}

func serverInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
    zlog.Log.Infof("Request to [ %v ]", info.FullMethod)

    // Calls the handler
    h, err := handler(ctx, req)

    return h, err
}
