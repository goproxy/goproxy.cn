package cfg

import (
	"errors"
	"flag"
	"fmt"
	stdlog "log"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/aofei/air"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

var (
	// a is the `air.Default`.
	a = air.Default

	// Zerolog is the Zerolog configuration items.
	Zerolog struct {
		// LoggerLevel is the logger level of the Zerolog.
		LoggerLevel string `mapstructure:"logger_level"`
	}

	// Qiniu is the Qiniu Cloud configuration items.
	Qiniu struct {
		// AccessKey is the access key of the Qiniu Cloud.
		AccessKey string `mapstructure:"access_key"`

		// SecretKey is the secret key of the Qiniu Cloud.
		SecretKey string `mapstructure:"secret_key"`

		// KodoEndpoint is the endpint of the Qiniu Cloud Kodo.
		KodoEndpoint string `mapstructure:"kodo_endpoint"`

		// KodoBucketName is the bucket name of the Qiniu Cloud Kodo.
		KodoBucketName string `mapstructure:"kodo_bucket_name"`

		// KodoBucketEndpoint is the bucket endpint of the Qiniu Cloud
		// Kodo.
		KodoBucketEndpoint string `mapstructure:"kodo_bucket_endpoint"`
	}

	// Goproxy is the Goproxy configuration items.
	Goproxy struct {
		// GoBinName is the name of the Go binary of the Goproxy.
		GoBinName string `mapstructure:"go_bin_name"`

		// MaxGoBinWorkers is the maximum number of the Go binary
		// commands that are allowed to execute at the same time of the
		// Goproxy.
		MaxGoBinWorkers int `mapstructure:"max_go_bin_workers"`

		// MaxZIPCacheBytes is the maximum number of bytes of the ZIP
		// cache that will be stored in the cacher of the Goproxy.
		MaxZIPCacheBytes int `mapstructure:"max_zip_cache_bytes"`

		// AutoRedirection indicates whether to control automatic
		// redirection of existing caches for the Goproxy.
		AutoRedirection bool `mapstructure:"auto_redirection"`

		// LocalCacheRoot is the root of the local caches of the
		// Goproxy.
		LocalCacheRoot string `mapstructure:"local_cache_root"`
	}
)

func init() {
	cf := flag.String("config", "config.toml", "configuration file")
	flag.Parse()

	m := map[string]interface{}{}
	if _, err := toml.DecodeFile(*cf, &m); err != nil {
		panic(fmt.Errorf(
			"failed to decode configuration file: %v",
			err,
		))
	}

	if err := mapstructure.Decode(m["air"], a); err != nil {
		panic(fmt.Errorf(
			"failed to decode air configuration items: %v",
			err,
		))
	}

	a.ErrorLogger = stdlog.New(&errorLogWriter{}, "", 0)

	if err := mapstructure.Decode(m["zerolog"], &Zerolog); err != nil {
		panic(fmt.Errorf(
			"failed to decode zerolog configuration items: %v",
			err,
		))
	}

	if err := mapstructure.Decode(m["qiniu"], &Qiniu); err != nil {
		panic(fmt.Errorf(
			"failed to decode qiniu configuration items: %v",
			err,
		))
	}

	if err := mapstructure.Decode(m["goproxy"], &Goproxy); err != nil {
		panic(fmt.Errorf(
			"failed to decode goproxy configuration items: %v",
			err,
		))
	}

	zerolog.TimeFieldFormat = ""
	switch Zerolog.LoggerLevel {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	case "fatal":
		zerolog.SetGlobalLevel(zerolog.FatalLevel)
	case "panic":
		zerolog.SetGlobalLevel(zerolog.PanicLevel)
	case "no":
		zerolog.SetGlobalLevel(zerolog.NoLevel)
	case "disabled":
		zerolog.SetGlobalLevel(zerolog.Disabled)
	}

	if a.DebugMode {
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	}
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
