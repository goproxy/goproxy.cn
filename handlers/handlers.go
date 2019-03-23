package handlers

import (
	"net/http"

	"github.com/aofei/air"
)

// a is the `air.Default`.
var a = air.Default

func init() {
	a.FILE("/robots.txt", "robots.txt")

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/",
		indexPageHandler,
	)
}

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	return res.Render(nil, "index.html")
}
