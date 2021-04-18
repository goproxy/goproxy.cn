package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/minio/minio-go/v7"
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
	objectInfo, err := qiniuKodoClient.StatObject(
		req.Context,
		qiniuKodoBucketName,
		"stats/summary",
		minio.StatObjectOptions{},
	)
	if err != nil {
		if isMinIOObjectNotExist(err) {
			return req.Air.NotFoundHandler(req, res)
		}

		return err
	}

	object, err := qiniuKodoClient.GetObject(
		req.Context,
		qiniuKodoBucketName,
		objectInfo.Key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	res.Header.Set("Content-Type", objectInfo.ContentType)
	res.Header.Set("ETag", fmt.Sprintf("%q", objectInfo.ETag))
	res.Header.Set(
		"Last-Modified",
		objectInfo.LastModified.UTC().Format(http.TimeFormat),
	)

	return res.Write(object)
}

// hStatTrend handles requests to query stat trend.
func hStatTrend(req *air.Request, res *air.Response) error {
	trend := req.ParamValue("Trend").String()
	switch trend {
	case "latest", "last-7-days", "last-30-days":
	default:
		return req.Air.NotFoundHandler(req, res)
	}

	objectInfo, err := qiniuKodoClient.StatObject(
		req.Context,
		qiniuKodoBucketName,
		fmt.Sprint("stats/trends/", trend),
		minio.StatObjectOptions{},
	)
	if err != nil {
		if isMinIOObjectNotExist(err) {
			return req.Air.NotFoundHandler(req, res)
		}

		return err
	}

	object, err := qiniuKodoClient.GetObject(
		req.Context,
		qiniuKodoBucketName,
		objectInfo.Key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	res.Header.Set("Content-Type", objectInfo.ContentType)
	res.Header.Set("ETag", fmt.Sprintf("%q", objectInfo.ETag))
	res.Header.Set(
		"Last-Modified",
		objectInfo.LastModified.UTC().Format(http.TimeFormat),
	)

	return res.Write(object)
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

	objectInfo, err := qiniuKodoClient.StatObject(
		req.Context,
		qiniuKodoBucketName,
		path.Join("stats", name),
		minio.StatObjectOptions{},
	)
	if err == nil {
		object, err := qiniuKodoClient.GetObject(
			req.Context,
			qiniuKodoBucketName,
			objectInfo.Key,
			minio.GetObjectOptions{},
		)
		if err != nil {
			return err
		}
		defer object.Close()

		if strings.HasSuffix(name, downloadCountBadgeSuffix) {
			res.Header.Set("Content-Type", objectInfo.ContentType)
			res.Header.Set(
				"ETag",
				fmt.Sprintf("%q", objectInfo.ETag),
			)
			res.Header.Set(
				"Last-Modified",
				objectInfo.LastModified.UTC().
					Format(http.TimeFormat),
			)

			return res.Write(object)
		}

		b, err := io.ReadAll(object)
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

		res.Header.Set("Content-Type", objectInfo.ContentType)

		return res.Write(bytes.NewReader(statJSON))
	} else if !isMinIOObjectNotExist(err) {
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
