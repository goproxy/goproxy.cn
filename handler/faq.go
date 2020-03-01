package handler

import (
	"fmt"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
)

func init() {
	base.Air.BATCH(getHeadMethods, "/faq", hFaqPage)
}

// hFaqPage handles requests to get FAQ page.
func hFaqPage(req *air.Request, res *air.Response) error {
	const faqPageURLBase = "https://github.com/goproxy/goproxy.cn/wiki/FAQ"
	return res.WriteHTML(fmt.Sprintf(
		"<meta http-equiv=refresh content=0;url=%s>",
		req.LocalizedString(faqPageURLBase),
	))
}
