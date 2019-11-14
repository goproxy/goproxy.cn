package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/air-gases/defibrillator"
	"github.com/air-gases/limiter"
	"github.com/air-gases/logger"
	"github.com/air-gases/redirector"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/cfg"
	_ "github.com/goproxy/goproxy.cn/handler"
	"github.com/rs/zerolog/log"
)

// a is the `air.Default`.
var a = air.Default

func main() {
	a.ErrorHandler = errorHandler

	a.Pregases = []air.Gas{
		logger.Gas(logger.GasConfig{
			IncludeClientAddress: true,
		}),
		defibrillator.Gas(defibrillator.GasConfig{}),
		redirector.WWW2NonWWWGas(redirector.WWW2NonWWWGasConfig{
			HTTPSEnforced: true,
		}),
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

	cfg.Cron.Start()
	<-shutdownChan
	<-cfg.Cron.Stop().Done()

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
