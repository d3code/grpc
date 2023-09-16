package server

import (
    "context"
    "github.com/d3code/zlog"
    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "net/http"
    "path"
)

type HttpGateway struct {
    Host            string
    Port            string
    GrpcConnections map[string]GrpcConnection
}

type GrpcConnection struct {
    Host         string
    Port         string
    GrpcHandlers []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
}

func (g *HttpGateway) Address() string {
    return g.Host + ":" + g.Port
}

func (i *GrpcConnection) Address() string {
    return i.Host + ":" + i.Port
}

// Run starts a HTTP gateway that serves the gRPC server
func (g *HttpGateway) Run(grpcHost string, grpcPort string) {

    // Create context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create HTTP handler
    mux := http.NewServeMux()

    // Create gRPC connections
    for p, grpcConnection := range g.GrpcConnections {

        // Dial the gRPC server
        conn, errDial := grpc.DialContext(ctx, grpcConnection.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
        if errDial != nil {
            zlog.Log.Fatalf("Failed to dial server: %v", errDial)
        }

        // Close the gRPC connection when the context is cancelled
        go closeDoneContextGrpcConnection(ctx, conn)

        x := runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
            pairs := metadata.Pairs("x-user-id", "1")
            return pairs
        })

        // Create gRPC handler
        gateway := runtime.NewServeMux(x)
        for _, grpcHandler := range grpcConnection.GrpcHandlers {
            errRegister := grpcHandler(ctx, gateway, conn)
            if errRegister != nil {
                return
            }
        }

        // Create Health handler
        health := serverHealth(conn)
        pattern := path.Join(p + "/health")
        mux.HandleFunc(pattern, health)

        mux.Handle(p, gateway)
    }

    // Add middleware
    handlerCORS := middlewareCORS(mux)
    handler := middlewareLog(handlerCORS)

    // Create HTTP server
    server := &http.Server{
        Addr:    g.Address(),
        Handler: handler,
    }

    // Close the HTTP server when the context is cancelled
    go closeDoneContextHttpServer(ctx, server)

    // Start the HTTP server
    zlog.Log.Infof("Starting HTTP server on %s", server.Addr)
    errListen := server.ListenAndServe()
    if errListen != http.ErrServerClosed {
        zlog.Log.Errorf("Failed to listen and serve: %v", errListen)
    }
}

func closeDoneContextHttpServer(ctx context.Context, server *http.Server) {
    <-ctx.Done()
    err := server.Shutdown(ctx)
    if err != nil {
        zlog.Log.Fatalf("Failed to shutdown HTTP server: %v", err)
    }
}

func closeDoneContextGrpcConnection(ctx context.Context, conn *grpc.ClientConn) {
    <-ctx.Done()
    err := conn.Close()
    if err != nil {
        zlog.Log.Fatalf("Failed to close connection to the gRPC server: %v", err)
    }
}
