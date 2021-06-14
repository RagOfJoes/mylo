package transport

import (
	"context"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

type HttpConfig struct {
	Host   string
	Port   string
	Scheme string

	RemoveExtraSlashes bool
}

type HttpResponse struct {
	Success bool             `json:"success"`
	Message string           `json:"message,omitempty"`
	Payload interface{}      `json:"payload,omitempty"`
	Error   *HttpClientError `json:"error,omitempty"`
}

// NewHTTP returns a configured gin engine instance
func NewHttp(cfg HttpConfig) *gin.Engine {
	ginEngine := gin.Default()
	ginEngine.RemoveExtraSlash = cfg.RemoveExtraSlashes
	return ginEngine
}

// RunHttp runs the http server with a graceful shutdown
// functionality
func RunHttp(cfg HttpConfig, g *gin.Engine) error {
	// Create a new http server from gin engine
	// instance
	addr := resolveAddr(cfg)
	srv := &http.Server{
		Addr:    addr,
		Handler: g,
	}
	// Setup graceful shutdown handling
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	// Initializing the server in a goroutine so that
	// it won't block the graceful shutdown handling below
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Listen: %s\n", err)
		}
	}()

	// Listen for the interrupt signal.
	<-ctx.Done()

	// Restore default behavior on the interrupt signal and notify user of shutdown.
	stop()
	log.Println("Shutting down gracefully, press Ctrl+C again to force")

	// The context is used to inform the server it has 5 seconds to finish
	// the request it is currently handling
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}

	log.Println("Server exiting")
	return nil
}

// Resolves address provided by http server
// configuration
func resolveAddr(cfg HttpConfig) string {
	if cfg.Port == "80" {
		return cfg.Host
	}
	if cfg.Host == ":" {
		return cfg.Host + cfg.Port

	}
	return cfg.Host + ":" + cfg.Port
}

// Retrieves request ip for logging purposes
func resolveIP(req *http.Request) string {
	real := req.Header.Get("X-Real-Ip")
	if len(real) > 0 {
		return real
	}
	forward := req.Header.Get("X-Forwarded-For")
	if len(forward) > 0 {
		return forward
	}
	return req.RemoteAddr
}
