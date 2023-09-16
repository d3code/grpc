package server

import (
    "context"
    "fmt"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "net"
)

func dial(ctx context.Context, network, addr string) (*grpc.ClientConn, error) {
    switch network {
    case "tcp":
        return dialTCP(ctx, addr)
    case "unix":
        return dialUnix(ctx, addr)
    default:
        return nil, fmt.Errorf("unsupported network type %q", network)
    }
}

func dialTCP(ctx context.Context, addr string) (*grpc.ClientConn, error) {
    return grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
}

func dialUnix(ctx context.Context, addr string) (*grpc.ClientConn, error) {
    d := func(ctx context.Context, addr string) (net.Conn, error) {
        return (&net.Dialer{}).DialContext(ctx, "unix", addr)
    }

    credentials := grpc.WithTransportCredentials(insecure.NewCredentials())
    return grpc.DialContext(ctx, addr, credentials, grpc.WithContextDialer(d))
}
