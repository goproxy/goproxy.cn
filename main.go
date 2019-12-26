package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/air-gases/defibrillator"
	"github.com/air-gases/limiter"
	"github.com/air-gases/logger"
	"github.com/air-gases/redirector"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/goproxy/goproxy.cn/handler"
)

func main() {
	base.Air.ErrorHandler = handler.Error
	base.Air.ErrorLogger = log.New(base.Logger, "", 0)

	base.Air.Pregases = []air.Gas{
		logger.Gas(logger.GasConfig{
			Logger:               &base.Logger,
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
		if err := base.Air.Serve(); err != nil {
			base.Logger.Error().Err(err).
				Msg("air server error")
		}
	}()

	<-shutdownChan

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	base.Air.Shutdown(ctx)
}
