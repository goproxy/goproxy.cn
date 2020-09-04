package base

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var (
	// Viper is the global instace of the `viper.Viper`.
	Viper = viper.New()

	// Logger is the global instace of the `zerolog.Logger`.
	Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Air is the global instace of the `air.Air`.
	Air = air.New()

	// Context is the global instance of the `context.Context`.
	Context context.Context

	// Cron is the global instance of the `cron.Cron`.
	Cron *cron.Cron
)

func init() {
	cf := pflag.StringP("config", "c", "config.toml", "configuration file")
	pflag.Parse()

	ext := filepath.Ext(*cf)
	Viper.AddConfigPath(filepath.Dir(*cf))
	Viper.SetConfigName(strings.TrimSuffix(filepath.Base(*cf), ext))
	Viper.SetConfigType(strings.TrimPrefix(ext, "."))
	if err := Viper.ReadInConfig(); err != nil {
		panic(fmt.Errorf("failed to read configuration file: %v", err))
	}

	zerolog.TimeFieldFormat = ""
	Logger = Logger.
		With().
		Str("app_name", Viper.GetString("air.app_name")).
		Logger()
	if Viper.GetBool("air.debug_mode") {
		Logger = Logger.Level(zerolog.DebugLevel)
	} else {
		l, _ := zerolog.ParseLevel(Viper.GetString("zerolog.level"))
		Logger = Logger.Level(l)
	}

	if err := Viper.UnmarshalKey("air", Air); err != nil {
		Logger.Fatal().Err(err).
			Msg("failed to unmarshal air configuration items")
	}

	var cancel context.CancelFunc
	Context, cancel = context.WithCancel(context.Background())
	Air.AddShutdownJob(cancel)

	Cron = cron.New(
		cron.WithLocation(time.UTC),
		cron.WithLogger(
			cron.PrintfLogger(log.New(Logger, "cron: ", 0)),
		),
	)
	Cron.Start()
	Air.AddShutdownJob(func() {
		<-Cron.Stop().Done()
	})
}
