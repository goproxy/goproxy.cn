package handler

import (
	"encoding/base64"
	"fmt"
	"path"
	"strings"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
)

func init() {
	base.Air.BATCH(
		getHeadMethods,
		"/stats/summary",
		hStatSummary,
		hourlyCachemanGas,
	)

	base.Air.BATCH(
		getHeadMethods,
		"/stats/trends/:Trend",
		hStatTrend,
		hourlyCachemanGas,
	)

	base.Air.BATCH(getHeadMethods, "/stats/*", hStat, hourlyCachemanGas)

	base.Air.BATCH(getHeadMethods, "/stats", hStatsPage)
}

// hStatSummary handles requests to query stat summary.
func hStatSummary(req *air.Request, res *air.Response) error {
	cache, err := qiniuKodoCacher.Cache(req.Context, "stats/summary")
	if err != nil {
		return err
	}
	defer cache.Close()

	res.Header.Set("Content-Type", cache.MIMEType())
	res.Header.Set("ETag", fmt.Sprintf(
		"%q",
		base64.StdEncoding.EncodeToString(cache.Checksum()),
	))

	return res.Write(cache)
}

// hStatTrend handles requests to query stat trend.
func hStatTrend(req *air.Request, res *air.Response) error {
	trend := req.Param("Trend").Value().String()
	switch trend {
	case "latest", "last-7-days", "last-30-days":
	default:
		return req.Air.NotFoundHandler(req, res)
	}

	cache, err := qiniuKodoCacher.Cache(
		req.Context,
		fmt.Sprint("stats/trends/", trend),
	)
	if err != nil {
		return err
	}
	defer cache.Close()

	res.Header.Set("Content-Type", cache.MIMEType())
	res.Header.Set("ETag", fmt.Sprintf(
		"%q",
		base64.StdEncoding.EncodeToString(cache.Checksum()),
	))

	return res.Write(cache)
}

// hStat handles requests to query stat.
func hStat(req *air.Request, res *air.Response) error {
	name := path.Clean(req.Param("*").Value().String())

	cache, err := qiniuKodoCacher.Cache(
		req.Context,
		path.Join("stats", name),
	)
	if err == nil {
		defer cache.Close()

		res.Header.Set("Content-Type", cache.MIMEType())
		res.Header.Set("ETag", fmt.Sprintf(
			"%q",
			base64.StdEncoding.EncodeToString(cache.Checksum()),
		))

		return res.Write(cache)
	} else if err != goproxy.ErrCacheNotFound {
		return err
	}

	switch {
	case strings.HasSuffix(name, "/badges/download-count.svg"):
		res.Header.Set("Content-Type", "image/svg+xml")
		return res.WriteFile("unknown-badge.svg")
	}

	res.Header.Set("Content-Type", "application/json; charset=utf-8")
	res.Header.Set("ETag", `"x9cdZUcPsO7jFB3h3qLvGwEMkStZGvg6vPd9PSQ8Hms="`)
	res.Header.Set("Last-Modified", "Thu, 26 Mar 2020 12:34:56 GMT")

	return res.Write(strings.NewReader(
		`{` +
			`"download_count":0,` +
			`"last_30_days":[],` +
			`"top_10_module_versions":[]` +
			`}`),
	)
}

// hStatsPage handles requests to get statistics page.
func hStatsPage(req *air.Request, res *air.Response) error {
	return res.Render(map[string]interface{}{
		"PageTitle":     req.LocalizedString("Statistics"),
		"CanonicalPath": "/stats",
		"IsStatsPage":   true,
	}, req.LocalizedString("stats.html"), "layouts/default.html")
}
