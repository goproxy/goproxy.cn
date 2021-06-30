package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/minio/minio-go/v7"
	"golang.org/x/mod/module"
)

// moduleVersionStat is the module version statastic.
type moduleVersionStat struct {
	DownloadCount int `json:"download_count"`
	Last30Days    []struct {
		Date          time.Time `json:"date"`
		DownloadCount int       `json:"download_count"`
	} `json:"last_30_days"`
	Top10ModuleVersions []struct {
		ModuleVersion string `json:"module_version"`
		DownloadCount int    `json:"download_count"`
	} `json:"top_10_module_versions,omitempty"`
}

// updateLast30Days updates `mvs.Last30Days` to the date.
func (mvs *moduleVersionStat) updateLast30Days(date time.Time) {
	last30Days := make([]struct {
		Date          time.Time `json:"date"`
		DownloadCount int       `json:"download_count"`
	}, 30)

	for i := 0; i < len(last30Days); i++ {
		last30Days[i].Date = date.AddDate(0, 0, -i)
		for _, d := range mvs.Last30Days {
			if d.Date == last30Days[i].Date {
				last30Days[i].DownloadCount = d.DownloadCount
				break
			}
		}
	}

	mvs.Last30Days = last30Days
}

func init() {
	base.Air.BATCH(
		getHeadMethods,
		"/stats/summary",
		hStatSummary,
		minutelyCachemanGas,
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
	object, err := qiniuKodoClient.GetObject(
		req.Context,
		qiniuKodoBucketName,
		"stats/summary",
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	objectInfo, err := object.Stat()
	if err != nil {
		if isNotFoundMinIOError(err) {
			return NotFound(req, res)
		}

		return err
	}

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
		return NotFound(req, res)
	}

	object, err := qiniuKodoClient.GetObject(
		req.Context,
		qiniuKodoBucketName,
		fmt.Sprint("stats/trends/", trend),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	objectInfo, err := object.Stat()
	if err != nil {
		if isNotFoundMinIOError(err) {
			return NotFound(req, res)
		}

		return err
	}

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

	name, err := url.PathUnescape(req.ParamValue("*").String())
	if err != nil || strings.HasSuffix(name, "/") {
		return CacheableNotFound(req, res, 86400)
	}

	if strings.Contains(name, "..") {
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return CacheableNotFound(req, res, 86400)
			}
		}
	}

	name = strings.TrimPrefix(path.Clean(name), "/")

	hasDownloadCountBadgeSuffix := strings.HasSuffix(
		name,
		downloadCountBadgeSuffix,
	)
	if hasDownloadCountBadgeSuffix {
		if module.CheckPath(strings.TrimSuffix(
			name,
			downloadCountBadgeSuffix,
		)) != nil {
			return CacheableNotFound(req, res, 86400)
		}
	} else if nameParts := strings.Split(name, "@"); len(nameParts) == 2 {
		if module.Check(nameParts[0], nameParts[1]) != nil {
			return CacheableNotFound(req, res, 86400)
		}
	} else if module.CheckPath(name) != nil {
		return CacheableNotFound(req, res, 86400)
	}

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

	object, err := qiniuKodoClient.GetObject(
		req.Context,
		qiniuKodoBucketName,
		path.Join("stats", name),
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	objectInfo, err := object.Stat()
	if err != nil {
		if isNotFoundMinIOError(err) {
			if hasDownloadCountBadgeSuffix {
				res.Header.Set("Content-Type", "image/svg+xml")
				return res.WriteFile("unknown-badge.svg")
			}

			var stat moduleVersionStat
			stat.updateLast30Days(date)

			statJSON, err := json.Marshal(stat)
			if err != nil {
				return err
			}

			res.Header.Set(
				"Content-Type",
				"application/json; charset=utf-8",
			)

			return res.Write(bytes.NewReader(statJSON))
		}

		return err
	}

	if hasDownloadCountBadgeSuffix {
		res.Header.Set("Content-Type", objectInfo.ContentType)
		res.Header.Set("ETag", fmt.Sprintf("%q", objectInfo.ETag))
		res.Header.Set(
			"Last-Modified",
			objectInfo.LastModified.UTC().Format(http.TimeFormat),
		)
		return res.Write(object)
	}

	b, err := io.ReadAll(object)
	if err != nil {
		return err
	}

	var stat moduleVersionStat
	if err := json.Unmarshal(b, &stat); err != nil {
		return err
	}

	stat.updateLast30Days(date)

	statJSON, err := json.Marshal(stat)
	if err != nil {
		return err
	}

	res.Header.Set("Content-Type", objectInfo.ContentType)

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
