package handler

import (
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
)

func init() {
	base.Air.BATCH(getHeadMethods, "/faq", hFaqPage)
}

// hFaqPage handles requests to get FAQ page.
func hFaqPage(req *air.Request, res *air.Response) error {
	return res.Render(map[string]interface{}{
		"PageTitle":     req.LocalizedString("FAQ"),
		"CanonicalPath": "/faq",
		"IsFAQPage":     true,
	}, req.LocalizedString("faq.html"), "layouts/default.html")
}
