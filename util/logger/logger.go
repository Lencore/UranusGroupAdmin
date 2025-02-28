package logger

import (
	"context"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger(env string, logLevel string) (*zap.Logger, error) {
	var ws []zapcore.WriteSyncer

	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	ws = append(ws, zapcore.AddSync(os.Stdout))

	level := zap.InfoLevel
	switch logLevel {
	case "debug":
		level = zap.DebugLevel
	case "info":
		level = zap.InfoLevel
	case "warn":
		level = zap.WarnLevel
	case "error":
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	if env == "dev" || env == "local" {
		level = zap.DebugLevel
	}

	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoder),
			zapcore.NewMultiWriteSyncer(ws...),
			level,
		),
	)

	zap.ReplaceGlobals(logger)

	return logger, nil
}

func FromContext(ctx context.Context) *zap.Logger {
	lg := zap.L()

	if requestID, ok := ctx.Value("requestid").(string); ok {
		lg = lg.With(zap.String("request_id", requestID))
	}

	return lg
}
