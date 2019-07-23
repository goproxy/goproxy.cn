package handler

import (
	"net/http"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/goproxy/goproxy/cacher"
)

// a is the `air.Default`.
var a = air.Default

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

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/",
		indexPageHandler,
	)
	a.BATCH(nil, "/*", air.WrapHTTPHandler(g))
}

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	return res.Redirect("https://github.com/goproxy/goproxy.cn")
}
