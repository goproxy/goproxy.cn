package main

import (
	"context"
	"errors"
	stdLog "log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/air-gases/defibrillator"
	"github.com/air-gases/limiter"
	"github.com/air-gases/logger"
	"github.com/air-gases/redirector"
	"github.com/aofei/air"
	_ "github.com/goproxy/goproxy.cn/handlers"
	"github.com/rs/zerolog/log"
)

// a is the `air.Default`.
var a = air.Default

func main() {
	a.ErrorHandler = errorHandler
	a.ErrorLogger = stdLog.New(&errorLogWriter{}, "", 0)

	a.Pregases = []air.Gas{
		logger.Gas(logger.GasConfig{}),
		defibrillator.Gas(defibrillator.GasConfig{}),
		redirector.WWW2NonWWWGas(redirector.WWW2NonWWWGasConfig{}),
		limiter.BodySizeGas(limiter.BodySizeGasConfig{
			MaxBytes: 1 << 20,
		}),
	}

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := a.Serve(); err != nil {
			log.Error().Err(err).
				Str("app_name", a.AppName).
				Msg("server error")
		}
	}()

	<-shutdownChan

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	a.Shutdown(ctx)
}

// errorHandler is the error handler.
func errorHandler(err error, req *air.Request, res *air.Response) {
	if res.Written {
		return
	}

	m := ""
	if !req.Air.DebugMode && res.Status == http.StatusInternalServerError {
		m = http.StatusText(res.Status)
	} else {
		m = err.Error()
	}

	res.WriteJSON(map[string]interface{}{
		"Error": m,
	})
}

// errorLogWriter is an error log writer.
type errorLogWriter struct{}

// Write implements the `io.Writer`.
func (elw *errorLogWriter) Write(b []byte) (int, error) {
	log.Error().Err(errors.New(strings.TrimSuffix(string(b), "\n"))).
		Str("app_name", a.AppName).
		Msg("air error")

	return len(b), nil
}
