package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"
	"unicode/utf8"

	"github.com/air-gases/defibrillator"
	"github.com/air-gases/langman"
	"github.com/air-gases/limiter"
	"github.com/air-gases/logger"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/goproxy/goproxy.cn/handler"
)

func main() {
	base.Air.NotFoundHandler = handler.NotFound
	base.Air.MethodNotAllowedHandler = handler.MethodNotAllowed
	base.Air.ErrorHandler = handler.Error
	base.Air.ErrorLogger = log.New(base.Logger, "", 0)

	base.Air.Pregases = []air.Gas{
		logger.Gas(logger.GasConfig{
			Logger:               &base.Logger,
			IncludeClientAddress: true,
		}),
		defibrillator.Gas(defibrillator.GasConfig{}),
		limiter.BodySizeGas(limiter.BodySizeGasConfig{
			MaxBytes: 1 << 20,
		}),
		func(next air.Handler) air.Handler {
			return func(req *air.Request, res *air.Response) error {
				path, err := url.PathUnescape(req.Path)
				if err != nil || !utf8.ValidString(path) {
					return handler.CacheableNotFound(
						req,
						res,
						86400,
					)
				}

				return next(req, res)
			}
		},
	}

	base.Air.Gases = []air.Gas{
		langman.Gas(langman.GasConfig{
			CookieMaxAge: 31536000,
		}),
	}

	go func() {
		if err := base.Air.Serve(); err != nil {
			base.Logger.Error().Err(err).
				Msg("air server error")
		}
	}()

	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)
	<-shutdownChan

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	base.Air.Shutdown(ctx)
}
