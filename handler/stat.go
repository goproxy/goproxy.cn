package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path"
	"strings"
	"time"

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

	date := time.Now().In(base.TZAsiaShanghai)
	if date.Hour() < 8 {
		date = time.Date(
			date.Year(),
			date.Month(),
			date.Day()-2,
			0,
			0,
			0,
			0,
			time.UTC,
		)
	} else {
		date = time.Date(
			date.Year(),
			date.Month(),
			date.Day()-1,
			0,
			0,
			0,
			0,
			time.UTC,
		)
	}

	mvs := struct {
		DownloadCount       int           `json:"download_count"`
		Last30Days          []interface{} `json:"last_30_days"`
		Top10ModuleVersions interface{}   `json:"top_10_module_versions,omitempty"`
	}{
		0,
		make([]interface{}, 30),
		nil,
	}

	for i := 0; i < len(mvs.Last30Days); i++ {
		date := date.AddDate(0, 0, -i)
		mvs.Last30Days[i] = struct {
			Date          string `json:"date"`
			DownloadCount int    `json:"download_count"`
		}{
			date.UTC().Format(time.RFC3339),
			0,
		}
	}

	if !strings.Contains(name, "@") {
		mvs.Top10ModuleVersions = make([]interface{}, 0)
	}

	statJSON, err := json.Marshal(&mvs)
	if err != nil {
		return err
	}

	res.Header.Set("Content-Type", "application/json; charset=utf-8")

	return res.Write(bytes.NewReader(statJSON))
}

// hStatsPage handles requests to get statistics page.
func hStatsPage(req *air.Request, res *air.Response) error {
	return res.Render(map[string]interface{}{
		"PageTitle":     req.LocalizedString("Statistics"),
		"CanonicalPath": "/stats",
		"IsStatsPage":   true,
	}, req.LocalizedString("stats.html"), "layouts/default.html")
}
