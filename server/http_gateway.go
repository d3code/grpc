package server

import (
    "context"
    "crypto/tls"
    "github.com/d3code/zlog"
    "github.com/google/uuid"
    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials"
    "google.golang.org/grpc/credentials/insecure"
    "google.golang.org/grpc/metadata"
    "google.golang.org/protobuf/proto"
    "net/http"
    "path"
)

const RequestIdHeaderName = "x-request-id"

type HttpGateway struct {
    Host            string
    Port            string
    GrpcConnections map[string]GrpcConnection
    HttpHandlers    map[string]http.Handler
}

type GrpcConnection struct {
    Host         string
    Port         string
    Secure       bool
    GrpcHandlers []func(context.Context, *runtime.ServeMux, *grpc.ClientConn) error
    HttpHandlers map[string]http.Handler
}

func (g *HttpGateway) Address() string {
    return g.Host + ":" + g.Port
}

func (i *GrpcConnection) Address() string {
    return i.Host + ":" + i.Port
}

// Run starts a HTTP gateway that serves the gRPC server
func (g *HttpGateway) Run() {

    // Create context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Create HTTP handler
    mux := http.NewServeMux()

    for p, httpHandler := range g.HttpHandlers {
        mux.Handle(p, removePathPrefix(p, httpHandler))
    }

    // Create gRPC connections
    for p, grpcConnection := range g.GrpcConnections {

        // Dial the gRPC server
        var connection *grpc.ClientConn
        if grpcConnection.Secure {
            zlog.Log.Warnf("Secure connection for path [%s] to %s", p, grpcConnection.Address())
            conn, errDial := grpc.DialContext(ctx, grpcConnection.Address(), grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{InsecureSkipVerify: true})))
            if errDial != nil {
                zlog.Log.Fatalf("Failed to dial server: %v", errDial)
                continue
            }
            connection = conn
        } else {
            zlog.Log.Warnf("Insecure connection for path [%s] to %s", p, grpcConnection.Address())
            conn, errDial := grpc.DialContext(ctx, grpcConnection.Address(), grpc.WithTransportCredentials(insecure.NewCredentials()))
            if errDial != nil {
                zlog.Log.Fatalf("Failed to dial server: %v", errDial)
                continue
            }
            connection = conn
        }

        // Close the gRPC connection when the context is cancelled
        go closeDoneContextGrpcConnection(ctx, connection)

        // Create metadata
        withMetadata := runtime.WithMetadata(func(ctx context.Context, req *http.Request) metadata.MD {
            pairs := metadata.Pairs(RequestIdHeaderName, uuid.New().String())
            return pairs
        })

        outgoingHeaderMatcher := runtime.WithOutgoingHeaderMatcher(func(key string) (string, bool) {
            return key, key != "content-type"
        })

        option := func(ctx context.Context, writer http.ResponseWriter, message proto.Message) error {
            if m, ok := metadata.FromOutgoingContext(ctx); ok {
                req := m.Get(RequestIdHeaderName)
                zlog.Log.Infof("Request ID: %v", req)
                if len(req) > 0 {
                    writer.Header().Set(RequestIdHeaderName, req[0])
                }
            }
            return nil
        }

        forwardResponseOption := runtime.WithForwardResponseOption(option)

        // Create gRPC handler
        gateway := runtime.NewServeMux(outgoingHeaderMatcher, forwardResponseOption, withMetadata)

        for _, grpcHandler := range grpcConnection.GrpcHandlers {
            errRegister := grpcHandler(ctx, gateway, connection)
            if errRegister != nil {
                return
            }
        }

        // Create Health handler
        health := serverHealth(connection)
        pattern := path.Join(p + "/health")
        mux.HandleFunc(pattern, health)

        //mux.Handle(p, gateway)
        mux.Handle(p, removePathPrefix(p, gateway))
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
