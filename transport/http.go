package transport

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/RagOfJoes/mylo/internal/config"
	"github.com/gin-gonic/gin"
)

// HttpErrorResponse defines the error structure that users will be able to see
type HttpErrorResponse struct {
	Title       string `json:"title"`
	Description string `json:"description"`
}

// HttpResponse defines the structure for responses that users will be able to see
type HttpResponse struct {
	Success bool               `json:"success"`
	Message string             `json:"message,omitempty"`
	Payload interface{}        `json:"payload,omitempty"`
	Error   *HttpErrorResponse `json:"error,omitempty"`
}

// NewHTTP returns a configured gin engine instance with some essential middlewares
func NewHttp() *gin.Engine {
	cfg := config.Get()
	ginEngine := gin.New()
	// LoggerWithFormatter middleware will write the logs to gin.DefaultWriter
	// By default gin.DefaultWriter = os.Stdout
	ginEngine.Use(gin.LoggerWithFormatter(
		func(param gin.LogFormatterParams) string {
			return fmt.Sprintf("[Request] %v |%s %3d %s| %13v | %15s |%s %-7s %s %s\n%s",
				param.TimeStamp.Format("2006/01/02 - 15:04:05"),
				param.StatusCodeColor(), param.StatusCode, param.ResetColor(),
				param.Latency,
				param.ClientIP,
				param.MethodColor(), param.Request.Method, param.ResetColor(),
				param.Request.URL.Path,
				param.ErrorMessage,
			)
		},
	))
	ginEngine.Use(gin.Recovery())

	ginEngine.RemoveExtraSlash = cfg.Server.ExtraSlash
	return ginEngine
}

// RunHttp runs the http server with a graceful shutdown
// functionality
func RunHttp(handler http.Handler) error {
	cfg := config.Get()
	// Create a new http server from gin engine
	// instance
	addr := resolveAddr(cfg.Server.Host, cfg.Server.Port)
	srv := &http.Server{
		Addr:    addr,
		Handler: handler,
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
func resolveAddr(host string, port int) string {
	if port == 80 {
		return host
	}
	if host == ":" {
		return fmt.Sprintf("%s%d", host, port)
	}
	return fmt.Sprintf("%s:%d", host, port)
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
