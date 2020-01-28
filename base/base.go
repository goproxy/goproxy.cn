package base

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aofei/air"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"
)

var (
	// Viper is the global instace of the `viper.Viper`.
	Viper = viper.New()

	// Logger is the global instace of the `zerolog.Logger`.
	Logger = zerolog.New(os.Stderr).With().Timestamp().Logger()

	// Air is the global instace of the `air.Air`.
	Air = air.New()
)

func init() {
	cf := flag.String("config", "config.toml", "configuration file")
	flag.Parse()

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
		ll, _ := zerolog.
			ParseLevel(Viper.GetString("zerolog.logger_level"))
		Logger = Logger.Level(ll)
	}

	if err := Viper.UnmarshalKey("air", Air); err != nil {
		Logger.Fatal().Err(err).
			Msg("failed to unmarshal air configuration items")
	}
}
