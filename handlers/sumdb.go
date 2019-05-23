package handlers

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/cfg"
	"golang.org/x/net/idna"
)

var supportedSUMDBHosts = map[string]bool{}

func init() {
	for _, host := range cfg.Goproxy.SupportedSUMDBHosts {
		if h, err := idna.Lookup.ToASCII(host); err == nil {
			supportedSUMDBHosts[h] = true
		}
	}

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/sumdb/*",
		sumdbHandler,
		cacheman.Gas(cacheman.GasConfig{
			MustRevalidate: true,
			NoCache:        true,
			NoStore:        true,
			MaxAge:         -1,
			SMaxAge:        -1,
		}),
	)
}

// sumdbHandler handles requests to perform a Go module proxy action for
// checksum database.
func sumdbHandler(req *air.Request, res *air.Response) error {
	sumdbURL := req.Param("*").Value().String()
	sumdbPathOffset := strings.Index(sumdbURL, "/")
	if sumdbPathOffset < 0 {
		return a.NotFoundHandler(req, res)
	}

	sumdbHost, err := idna.Lookup.ToASCII(sumdbURL[:sumdbPathOffset])
	if err != nil {
		return a.NotFoundHandler(req, res)
	}

	if !supportedSUMDBHosts[sumdbHost] {
		return a.NotFoundHandler(req, res)
	}

	sumdbPath := sumdbURL[sumdbPathOffset:]
	if sumdbPath == "/supported" {
		res.Status = http.StatusOK
		return res.Write(nil)
	}

	sumdbRes, err := http.Get(fmt.Sprint("https://", sumdbHost, sumdbPath))
	if err != nil {
		return err
	}
	defer sumdbRes.Body.Close()

	if sumdbRes.StatusCode != http.StatusOK {
		return a.NotFoundHandler(req, res)
	}

	res.Header.Set("Content-Type", sumdbRes.Header.Get("Content-Type"))
	res.Header.Set(
		"Content-Length",
		strconv.FormatInt(sumdbRes.ContentLength, 10),
	)

	_, err = io.Copy(res.Body, sumdbRes.Body)

	return err
}
