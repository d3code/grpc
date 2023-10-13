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
    PreRequest       func(ctx context.Context, req any, info *grpc.UnaryServerInfo) (err error)
    PostRequest      func(ctx context.Context, req any, info *grpc.UnaryServerInfo) (err error)
}

func (s *GrpcServer) Address() string {
    return s.Host + ":" + s.Port
}

func (s *GrpcServer) Run() {
    listen, err := net.Listen("tcp", s.Address())
    if err != nil {
        zlog.Log.Fatalf("Failed to listen: %s", err)
        return
    }

    x := func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
        zlog.Log.Infof("gRPC Request [ %v ]", info.FullMethod)

        if s.PreRequest != nil {
            err = s.PreRequest(ctx, req, info)
            if err != nil {
                zlog.Log.Error(err)
                return resp, err
            }
        }

        h, err := handler(ctx, req)

        if err != nil {
            zlog.Log.Error(err)
            return h, err
        }

        if s.PostRequest != nil {
            err = s.PostRequest(ctx, req, info)
            if err != nil {
                zlog.Log.Error(err)
            }
        }

        return h, err
    }

    // Create gRPC server
    interceptor := grpc.UnaryInterceptor(x)
    server := grpc.NewServer(interceptor)
    reflection.Register(server)

    // Register services
    s.RegisterServices(server)

    if listen == nil || listen.Addr() == nil || server == nil {
        zlog.Log.Fatalf("Failed to listen or create server")
        return
    }

    // Start gRPC server
    zlog.Log.Infof("Starting gRPC server on %s", listen.Addr().String())
    err = server.Serve(listen)
    if err != nil {
        zlog.Log.Fatalf("Failed to serve: %s", err)
    }
}
