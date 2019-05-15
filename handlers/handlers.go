package handlers

import (
	"net/http"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"github.com/rs/zerolog/log"
)

var (
	// a is the `air.Default`.
	a = air.Default

	qiniuMac                  *qbox.Mac
	qiniuStorageConfig        *storage.Config
	qiniuStorageBucketManager *storage.BucketManager
)

func init() {
	qiniuMac = qbox.NewMac(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)

	qiniuStorageRegion, err := storage.GetRegion(
		cfg.Qiniu.AccessKey,
		cfg.Qiniu.StorageBucket,
	)
	if err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to get qiniu storage region client")
	}

	qiniuStorageConfig = &storage.Config{
		Region: qiniuStorageRegion,
	}

	qiniuStorageBucketManager = storage.NewBucketManager(
		qiniuMac,
		qiniuStorageConfig,
	)

	a.FILE("/robots.txt", "robots.txt")

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/",
		indexPageHandler,
	)
}

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	// return res.Render(nil, "index.html")
	return res.Redirect("https://github.com/goproxy/goproxy.cn")
}
