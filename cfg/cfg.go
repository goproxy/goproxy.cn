package cfg

import (
	"errors"
	"flag"
	"fmt"
	stdLog "log"
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

	// Goproxy is the Goproxy configuration items.
	Goproxy struct {
		// GoBinName is the name of the Go binary of the Goproxy.
		GoBinName string `mapstructure:"go_bin_name"`

		// MaxGoBinWorkers is the maximum number of the Go binary
		// commands of the Goproxy that are allowed to execute at the
		// same time.
		MaxGoBinWorkers int `mapstructure:"max_go_bin_workers"`

		// SupportedSUMDBHosts is the supported checksum database host
		// of the Goproxy.
		SupportedSUMDBHosts []string `mapstructure:"supported_sumdb_hosts"`

		// KodoEndpoint is the endpoint of the Qiniu Cloud Kodo.
		KodoEndpoint string `mapstructure:"kodo_endpoint"`

		// KodoAccessKey is the access key of the Qiniu Cloud Kodo.
		KodoAccessKey string `mapstructure:"kodo_access_key"`

		// KodoSecretKey is the secret key of the Qiniu Cloud Kodo.
		KodoSecretKey string `mapstructure:"kodo_secret_key"`

		// KodoBucketName is the bucket name of the Qiniu Cloud Kodo.
		KodoBucketName string `mapstructure:"kodo_bucket_name"`
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

	a.ErrorLogger = stdLog.New(&errorLogWriter{}, "", 0)

	if err := mapstructure.Decode(m["zerolog"], &Zerolog); err != nil {
		panic(fmt.Errorf(
			"failed to decode zerolog configuration items: %v",
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
