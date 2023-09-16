package server

import (
    "encoding/json"
    "github.com/d3code/zlog"
    "github.com/golang/glog"
    "google.golang.org/grpc"
    "net/http"
    "path"
    "strings"
)

type Health struct {
    Status     string `json:"status"`
    Connection string `json:"connection"`
}

func serverHealth(conn *grpc.ClientConn) http.HandlerFunc {
    handler := func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")

        status := Health{
            Status:     conn.GetState().String(),
            Connection: conn.Target(),
        }

        resp, err := json.Marshal(status)
        if err != nil {
            zlog.Log.Errorf("Failed to marshal response: %v", err)
            return
        }

        _, err = w.Write(resp)
        if err != nil {
            zlog.Log.Errorf("Failed to write response: %v %s", err)
            return
        }
    }
    return handler
}

func serverOpenAPI(dir string) http.HandlerFunc {
    handler := func(w http.ResponseWriter, r *http.Request) {
        if !strings.HasSuffix(r.URL.Path, ".swagger.json") {
            glog.Errorf("Not Found: %s", r.URL.Path)
            http.NotFound(w, r)
            return
        }
        zlog.Log.Infof("Serving %s", r.URL.Path)
        p := strings.TrimPrefix(r.URL.Path, "/openapi/")
        p = path.Join(dir, p)
        http.ServeFile(w, r, p)
    }
    return handler
}
