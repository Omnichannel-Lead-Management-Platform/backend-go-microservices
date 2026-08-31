package logger

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log *zap.Logger

// Init initializes the global logger for the given service name.
func Init(serviceName string) error {
	// Setup lumberjack for log rolling
	logPath := fmt.Sprintf("/app/logs/%s.log", serviceName)
	if os.Getenv("ENV") == "development" || os.Getenv("ENV") == "" {
		// Local fallback if running outside docker without /app/logs
		logPath = fmt.Sprintf("./logs/%s.log", serviceName)
	}

	fileWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // megabytes
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}

	// Console encoder for human readable output
	consoleEncoder := zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig())

	// JSON encoder for file output (structured logging)
	productionConfig := zap.NewProductionEncoderConfig()
	productionConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	jsonEncoder := zapcore.NewJSONEncoder(productionConfig)

	// We want to write to both console and file
	core := zapcore.NewTee(
		zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), zap.DebugLevel),
		zapcore.NewCore(jsonEncoder, zapcore.AddSync(fileWriter), zap.InfoLevel),
	)

	Log = zap.New(core, zap.AddCaller(), zap.Fields(zap.String("service", serviceName)))
	
	// Replace global zap logger
	zap.ReplaceGlobals(Log)

	return nil
}

// ChiLogger is a custom logging middleware for Chi router using Zap
func ChiLogger(l *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			t1 := time.Now()

			defer func() {
				l.Info("Served",
					zap.String("method", r.Method),
					zap.String("path", r.URL.Path),
					zap.Int("status", ww.Status()),
					zap.Int("size", ww.BytesWritten()),
					zap.Duration("latency", time.Since(t1)),
					zap.String("reqId", middleware.GetReqID(r.Context())),
				)
			}()

			next.ServeHTTP(ww, r)
		}
		return http.HandlerFunc(fn)
	}
}

// AuthbossLoggerAdapter adapts zap to the standard library log.Logger used by Authboss
type AuthbossLoggerAdapter struct {
	logger *zap.Logger
}

func NewAuthbossLoggerAdapter(l *zap.Logger) *AuthbossLoggerAdapter {
	return &AuthbossLoggerAdapter{logger: l}
}

func (a *AuthbossLoggerAdapter) Info(arg string) {
	a.logger.Info(arg)
}

func (a *AuthbossLoggerAdapter) Error(arg string) {
	a.logger.Error(arg)
}
