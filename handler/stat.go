package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/tidwall/gjson"
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
	res.Header.Set(
		"Last-Modified",
		cache.ModTime().UTC().Format(http.TimeFormat),
	)

	return res.Write(cache)
}

// hStatTrend handles requests to query stat trend.
func hStatTrend(req *air.Request, res *air.Response) error {
	trend := req.ParamValue("Trend").String()
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
	res.Header.Set(
		"Last-Modified",
		cache.ModTime().UTC().Format(http.TimeFormat),
	)

	return res.Write(cache)
}

// hStat handles requests to query stat.
func hStat(req *air.Request, res *air.Response) error {
	const downloadCountBadgeSuffix = "/badges/download-count.svg"

	name := path.Clean(req.ParamValue("*").String())

	date := time.Now().UTC()
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

	cache, err := qiniuKodoCacher.Cache(
		req.Context,
		path.Join("stats", name),
	)
	if err == nil {
		defer cache.Close()

		if strings.HasSuffix(name, downloadCountBadgeSuffix) {
			res.Header.Set("Content-Type", cache.MIMEType())
			res.Header.Set("ETag", fmt.Sprintf(
				"%q",
				base64.StdEncoding.
					EncodeToString(cache.Checksum()),
			))
			res.Header.Set(
				"Last-Modified",
				cache.ModTime().UTC().Format(http.TimeFormat),
			)

			return res.Write(cache)
		}

		b, err := ioutil.ReadAll(cache)
		if err != nil {
			return err
		}

		gjr := gjson.ParseBytes(b)

		mvs := struct {
			DownloadCount       int64         `json:"download_count"`
			Last30Days          []interface{} `json:"last_30_days"`
			Top10ModuleVersions interface{}   `json:"top_10_module_versions,omitempty"`
		}{
			gjr.Get("download_count").Int(),
			make([]interface{}, 30),
			gjr.Get("top_10_module_versions").Value(),
		}

		gjrL30DsArray := gjr.Get("last_30_days").Array()
		for i := 0; i < len(mvs.Last30Days); i++ {
			date := date.AddDate(0, 0, -i).Format(time.RFC3339)
			var downloadCount int64
			for _, gjr := range gjrL30DsArray {
				if gjr.Get("date").String() != date {
					continue
				}

				downloadCount = gjr.Get("download_count").Int()
			}

			mvs.Last30Days[i] = struct {
				Date          string `json:"date"`
				DownloadCount int64  `json:"download_count"`
			}{
				date,
				downloadCount,
			}
		}

		statJSON, err := json.Marshal(mvs)
		if err != nil {
			return err
		}

		res.Header.Set("Content-Type", cache.MIMEType())

		return res.Write(bytes.NewReader(statJSON))
	} else if err != goproxy.ErrCacheNotFound {
		return err
	}

	switch {
	case strings.HasSuffix(name, downloadCountBadgeSuffix):
		res.Header.Set("Content-Type", "image/svg+xml")
		return res.WriteFile("unknown-badge.svg")
	}

	mvs := struct {
		DownloadCount       int64         `json:"download_count"`
		Last30Days          []interface{} `json:"last_30_days"`
		Top10ModuleVersions interface{}   `json:"top_10_module_versions,omitempty"`
	}{
		0,
		make([]interface{}, 30),
		nil,
	}

	for i := 0; i < len(mvs.Last30Days); i++ {
		mvs.Last30Days[i] = struct {
			Date          string `json:"date"`
			DownloadCount int64  `json:"download_count"`
		}{
			date.AddDate(0, 0, -i).Format(time.RFC3339),
			0,
		}
	}

	if !strings.Contains(name, "@") {
		mvs.Top10ModuleVersions = make([]interface{}, 0)
	}

	statJSON, err := json.Marshal(mvs)
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
