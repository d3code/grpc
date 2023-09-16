package server

import (
    "github.com/d3code/zlog"
    "net/http"
    "strings"
    "time"
)

// middlewareLog logs the request URI and the time it took to process the request
func middlewareLog(next http.Handler) http.Handler {
    handler := func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        next.ServeHTTP(w, r)

        elapsed := time.Since(start)
        zlog.Log.Infof("Request to [ %s ] processed in %s", r.RequestURI, elapsed)
    }
    return http.HandlerFunc(handler)
}

// middlewareCORS is a middlewareLog that adds CORS headers to the response if the Origin header is set
func middlewareCORS(h http.Handler) http.Handler {
    interceptor := func(w http.ResponseWriter, r *http.Request) {
        if origin := r.Header.Get("Origin"); origin != "" {

            w.Header().Set("Access-Control-Allow-Origin", origin)

            if r.Method == "OPTIONS" && r.Header.Get("Access-Control-Request-Method") != "" {
                headers := []string{"Content-Type", "Accept", "Authorization"}
                methods := []string{"GET", "HEAD", "POST", "PUT", "DELETE"}

                w.Header().Set("Access-Control-Allow-Headers", strings.Join(headers, ","))
                w.Header().Set("Access-Control-Allow-Methods", strings.Join(methods, ","))

                zlog.Log.Infof("Preflight request for %s", r.URL.Path)
                return
            }
        }
        h.ServeHTTP(w, r)
    }

    return http.HandlerFunc(interceptor)
}
