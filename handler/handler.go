package handler

import (
	"net/http"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/goproxy/goproxy/cacher"
)

var (
	// a is the `air.Default`.
	a = air.Default

	// getHeadMethods is an array contains the GET and the HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// cachemanGas is used to manage the Cache-Control header.
	cachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})
)

func init() {
	g := goproxy.New()
	g.GoBinName = cfg.Goproxy.GoBinName
	g.Cacher = &cacher.Kodo{
		Endpoint:   cfg.Goproxy.KodoEndpoint,
		AccessKey:  cfg.Goproxy.KodoAccessKey,
		SecretKey:  cfg.Goproxy.KodoSecretKey,
		BucketName: cfg.Goproxy.KodoBucketName,
	}

	g.ErrorLogger = a.ErrorLogger

	a.FILE("/robots.txt", "robots.txt")
	a.FILE("/favicon.ico", "favicon.ico", cachemanGas)
	a.FILE("/apple-touch-icon.png", "apple-touch-icon.png", cachemanGas)
	a.FILES("/assets", a.CofferAssetRoot, cachemanGas)
	a.BATCH(getHeadMethods, "/", indexPageHandler, cachemanGas)
	a.BATCH(nil, "/*", air.WrapHTTPHandler(g))
}

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	return res.Redirect("https://github.com/goproxy/goproxy.cn")
}
