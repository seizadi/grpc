package main

import (
  "flag"
  "fmt"
  "log"
  "net/http"
  "net/textproto"
  "os"
  "os/signal"
  "strings"
  "time"

  "github.com/grpc-ecosystem/grpc-gateway/runtime"
  "github.com/spf13/pflag"
  "github.com/spf13/viper"
  "golang.org/x/net/context"
  "google.golang.org/grpc"

  gw "gen/pb-go"
)

type proxyConfig struct {
  // The backend gRPC service to listen to.
  backend              string
  // Path to the swagger file to serve.
  swagger              string
  // Value to set for Access-Control-Allow-Origin header.
  corsAllowOrigin      string
  // Value to set for Access-Control-Allow-Credentials header.
  corsAllowCredentials string
  // Value to set for Access-Control-Allow-Methods header.
  corsAllowMethods string
  // Value to set for Access-Control-Allow-Headers header.
  corsAllowHeaders string
  // Prefix that this gateway is running on. For example, if your API endpoint
  // was "/foo/bar" in your protofile, and you wanted to run APIs under "/api",
  // set this to "/api/".
  apiPrefix            string
}

func allowCors(cfg proxyConfig, handler http.Handler) http.Handler {
  return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
    corsAllowOrigin := cfg.corsAllowOrigin
    if corsAllowOrigin == "*" {
      if origin := req.Header.Get("Origin"); origin != "" {
        corsAllowOrigin = origin
      }
    }
    w.Header().Set("Access-Control-Allow-Origin", corsAllowOrigin)
    w.Header().Set("Access-Control-Allow-Credentials", cfg.corsAllowCredentials)
    w.Header().Set("Access-Control-Allow-Methods", cfg.corsAllowMethods)
    w.Header().Set("Access-Control-Allow-Headers", cfg.corsAllowHeaders)
    if req.Method == "OPTIONS" && req.Header.Get("Access-Control-Request-Method") != "" {
      return
    }
    handler.ServeHTTP(w, req)
  })
}

// sanitizeApiPrefix forces prefix to be non-empty and end with a slash.
func sanitizeApiPrefix(prefix string) string {
  if len(prefix) == 0 || prefix[len(prefix)-1:] != "/" {
    return prefix + "/"
  }
  return prefix
}

// isPermanentHTTPHeader checks whether hdr belongs to the list of
// permenant request headers maintained by IANA.
// http://www.iana.org/assignments/message-headers/message-headers.xml
// From https://github.com/grpc-ecosystem/grpc-gateway/blob/7a2a43655ccd9a488d423ea41a3fc723af103eda/runtime/context.go#L157
func isPermanentHTTPHeader(hdr string) bool {
	switch hdr {
	case
		"Accept",
		"Accept-Charset",
		"Accept-Language",
		"Accept-Ranges",
		"Authorization",
		"Cache-Control",
		"Content-Type",
		"Cookie",
		"Date",
		"Expect",
		"From",
		"Host",
		"If-Match",
		"If-Modified-Since",
		"If-None-Match",
		"If-Schedule-Tag-Match",
		"If-Unmodified-Since",
		"Max-Forwards",
		"Origin",
		"Pragma",
		"Referer",
		"User-Agent",
		"Via",
		"Warning":
		return true
	}
	return false
}

// isReserved returns whether the key is reserved by gRPC.
func isReserved(key string) bool {
  return strings.HasPrefix(key, "Grpc-")
}

// incomingHeaderMatcher converts an HTTP header name on http.Request to
// grpc metadata. Permanent headers (i.e. User-Agent) are prepended with
// "grpc-gateway". Headers that start with start with "Grpc-" (reserved
// by grpc) are prepended with "X-". Other headers are forwarded as is.
func incomingHeaderMatcher(key string) (string, bool) {
  key = textproto.CanonicalMIMEHeaderKey(key)
  if isPermanentHTTPHeader(key) {
    return runtime.MetadataPrefix + key, true
  }
  if isReserved(key) {
    return "X-" + key, true
  }

  // The Istio service mesh dislikes when you pass the Content-Length header
  if key == "Content-Length" {
    return "", false
  }

  return key, true
}

// outgoingHeaderMatcher transforms outgoing metadata into HTTP headers.
// We return any response metadata as is.
func outgoingHeaderMatcher(metadata string) (string, bool) {
  return metadata, true
}

func SetupMux(ctx context.Context, cfg proxyConfig) *http.ServeMux {
  log.Printf("Creating grpc-gateway proxy with config: %v", cfg)
  mux := http.NewServeMux()

  mux.HandleFunc("/swagger.json", func(w http.ResponseWriter, r *http.Request) {
    http.ServeFile(w, r, cfg.swagger)
  })

  gwmux := runtime.NewServeMux(
    runtime.WithIncomingHeaderMatcher(incomingHeaderMatcher),
    runtime.WithOutgoingHeaderMatcher(outgoingHeaderMatcher),
  )
  fmt.Printf("Proxying requests to gRPC service at '%s'\n", cfg.backend)

  opts := []grpc.DialOption{grpc.WithInsecure()}
  // If you get a compilation error that gw.RegisterStarfriendsHandlerFromEndpoint
  // does not exist, it's because you haven't added any google.api.http annotations
  // to your proto. Add some!
  err := gw.RegisterStarfriendsHandlerFromEndpoint(ctx, gwmux, cfg.backend, opts)
  if err != nil {
    log.Fatalf("Could not register gateway: %v", err)
  }

  prefix := sanitizeApiPrefix(cfg.apiPrefix)
  log.Println("API prefix is", prefix)
  mux.Handle(prefix, http.StripPrefix(prefix[:len(prefix)-1], allowCors(cfg, gwmux)))
  return mux
}

// SetupViper returns a viper configuration object
func SetupViper() *viper.Viper {
  viper.SetConfigName("config")
  viper.AddConfigPath(".")
  viper.SetEnvPrefix("Starfriends")
  viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
  viper.AutomaticEnv()

  flag.String("backend", "", "The gRPC backend service to proxy.")

  pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
  pflag.Parse()
  viper.BindPFlags(pflag.CommandLine)

  err := viper.ReadInConfig()
  if err != nil {
    log.Fatalf("Could not read config: %v", err)
  }

  return viper.GetViper()
}

// SignalRunner runs a runner function until an interrupt signal is received, at which point it
// will call stopper.
func SignalRunner(runner, stopper func()) {
  signals := make(chan os.Signal, 1)
  signal.Notify(signals, os.Interrupt, os.Kill)

  go func() {
    runner()
  }()

  fmt.Println("hit Ctrl-C to shutdown")
  select {
  case <-signals:
    stopper()
  }
}

func main() {

  cfg := SetupViper()
  ctx := context.Background()
  ctx, cancel := context.WithCancel(ctx)
  defer cancel()

  mux := SetupMux(ctx, proxyConfig{
    backend:              cfg.GetString("backend"),
    swagger:              cfg.GetString("swagger.file"),
    corsAllowOrigin:      cfg.GetString("cors.allow-origin"),
    corsAllowCredentials: cfg.GetString("cors.allow-credentials"),
    corsAllowMethods:     cfg.GetString("cors.allow-methods"),
    corsAllowHeaders:     cfg.GetString("cors.allow-headers"),
    apiPrefix:            cfg.GetString("proxy.api-prefix"),
  })

  addr := fmt.Sprintf(":%v", cfg.GetInt("proxy.port"))
  server := &http.Server{Addr: addr, Handler: mux}

  SignalRunner(
    func() {
      fmt.Printf("launching http server on %v\n", server.Addr)
      if err := server.ListenAndServe(); err != nil {
        log.Fatalf("Could not start http server: %v", err)
      }
    },
    func() {
      shutdown, _ := context.WithTimeout(ctx, 10*time.Second)
      server.Shutdown(shutdown)
    })
}
