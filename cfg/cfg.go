package cfg

import (
	"flag"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/aofei/air"
	"github.com/mitchellh/mapstructure"
	"github.com/rs/zerolog"
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

		// MaxLocalCacheBytes is the maximum number of bytes of the
		// local cache the Goproxy will use.
		MaxLocalCacheBytes int `mapstructure:"max_local_cache_bytes"`

		// QiniuAccessKey is the access key of the Qiniu of the Goproxy.
		QiniuAccessKey string `mapstructure:"qiniu_access_key"`

		// QiniuSecretKey is the secret key of the Qiniu of the Goproxy.
		QiniuSecretKey string `mapstructure:"qiniu_secret_key"`

		// QiniuStorageBucket is the storage bucket of the Qiniu of the
		// Goproxy.
		QiniuStorageBucket string `mapstructure:"qiniu_storage_bucket"`

		// QiniuStorageBucketAccessEndpoint is the storage bucket access
		// endpoint of the Qiniu of the Goproxy.
		QiniuStorageBucketAccessEndpoint string `mapstructure:"qiniu_storage_bucket_access_endpoint"`
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
