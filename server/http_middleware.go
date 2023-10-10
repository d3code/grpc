package server

import (
    "github.com/d3code/zlog"
    "github.com/google/uuid"
    "net/http"
    "strings"
    "time"
)

// middlewareLog logs the request URI and the time it took to process the request
func middlewareLog(next http.Handler) http.Handler {
    handler := func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()

        requestId := uuid.New().String()
        r.Header.Set("X-Request-Id", requestId)

        zlog.Log.Infow("Request received [ "+r.RequestURI+" ]",
            "Request ID", requestId,
            "method", r.Method,
            "uri", r.RequestURI,
            "protocol", r.Proto,
            "host", r.Host,
            "accept", r.Header.Get("Accept"),
            "accept-encoding", r.Header.Get("Accept-Encoding"),
            "accept-language", r.Header.Get("Accept-Language"),
            "content-type", r.Header.Get("Content-Type"),
            "authorization", r.Header.Get("Authorization"),
            "content-length", r.ContentLength,
            "remote-address", r.RemoteAddr,
            "user-agent", r.UserAgent(),
            "referer", r.Referer())

        next.ServeHTTP(w, r)

        elapsed := time.Since(start)
        zlog.Log.Infof("Request ID [ %s ] processed in %s", requestId, elapsed)
    }
    return http.HandlerFunc(handler)
}

// removePathPrefix removes the prefix from the request URL
func removePathPrefix(prefix string, next http.Handler) http.Handler {
    handler := func(w http.ResponseWriter, r *http.Request) {
        r.URL.Path = r.URL.Path[len(prefix)-1:]
        next.ServeHTTP(w, r)
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

                zlog.Log.Infof("Preflight request for [ %s ]", r.URL.Path)
                return
            }
        }
        h.ServeHTTP(w, r)
    }

    return http.HandlerFunc(interceptor)
}
