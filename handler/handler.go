package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/tidwall/gjson"
)

var (
	// qiniuViper is used to get the configuration items of the Qiniu Cloud.
	qiniuViper = base.Viper.Sub("qiniu")

	// qiniuAccessKey is the access key for the Qiniu Cloud.
	qiniuAccessKey = qiniuViper.GetString("access_key")

	// qiniuSecretKey is the secret key for the Qiniu Cloud.
	qiniuSecretKey = qiniuViper.GetString("secret_key")

	// getHeadMethods is an array contains the GET and the HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// hourlyCachemanGas is used to manage the Cache-Control header.
	hourlyCachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})

	// cachedModuleVersionCount is the cached module version count.
	cachedModuleVersionCount int64
)

func init() {
	updateCachedModuleVersionsCount()
	if cachedModuleVersionCount == 0 {
		base.Logger.Fatal().
			Msg("failed to initialize cached module version count")
	}

	if _, err := base.Cron.AddFunc(
		"0 0 * * * *", // every 1 hour
		updateCachedModuleVersionsCount,
	); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to add cached module version count " +
				"update cron job")
	}

	base.Air.FILE("/robots.txt", "robots.txt")
	base.Air.FILE("/favicon.ico", "favicon.ico", hourlyCachemanGas)
	base.Air.FILE(
		"/apple-touch-icon.png",
		"apple-touch-icon.png",
		hourlyCachemanGas,
	)

	base.Air.FILES("/assets", base.Air.CofferAssetRoot, hourlyCachemanGas)

	base.Air.BATCH(getHeadMethods, "/", hIndexPage)
}

// Error handles errors.
func Error(err error, req *air.Request, res *air.Response) {
	if res.Written {
		return
	}

	m := ""
	if !req.Air.DebugMode && res.Status == http.StatusInternalServerError {
		m = http.StatusText(res.Status)
	} else {
		m = err.Error()
	}

	res.WriteJSON(map[string]interface{}{
		"Error": m,
	})
}

// hIndexPage handles requests to get index page.
func hIndexPage(req *air.Request, res *air.Response) error {
	return res.Render(map[string]interface{}{
		"IsIndexPage": true,
		"CachedModuleVersionCount": thousandsCommaSeperated(
			cachedModuleVersionCount,
		),
	}, req.LocalizedString("index.html"), "layouts/default.html")
}

// updateCachedModuleVersionsCount updates the `cachedModuleVersionCount`.
func updateCachedModuleVersionsCount() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	defer base.Air.RemoveShutdownJob(base.Air.AddShutdownJob(cancel))

	b, err := requestQiniuAPI(
		ctx,
		http.MethodGet,
		fmt.Sprintf(
			"https://api.qiniu.com"+
				"/v6/count?bucket=%s&begin=%s&end=%s&g=day",
			qiniuViper.GetString("kodo_bucket_name"),
			time.Now().Add(-time.Hour).In(base.TZAsiaShanghai).
				Format("20060102150405"),
			time.Now().In(base.TZAsiaShanghai).
				Format("20060102150405"),
		),
		"",
		nil,
	)
	if err != nil {
		base.Logger.Error().Err(err).
			Msg("failed to update cached module version count")
	}

	count := gjson.GetBytes(b, "datas.0").Int()
	if count > 0 {
		cachedModuleVersionCount = count / 3
	}
}

// requestQiniuAPI requests Qiniu API.
func requestQiniuAPI(
	ctx context.Context,
	method string,
	url string,
	contentType string,
	body io.Reader,
) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	path := req.URL.Path
	if req.URL.RawQuery != "" {
		path = fmt.Sprint(path, "?", req.URL.RawQuery)
	}

	data := []byte(fmt.Sprint(path, "\n"))
	if contentType == "application/x-www-form-urlencoded" && body != nil {
		b, err := ioutil.ReadAll(body)
		if err != nil {
			return nil, err
		}

		data = append(data, b...)

		req.Body = ioutil.NopCloser(bytes.NewReader(b))
	}

	h := hmac.New(sha1.New, []byte(qiniuSecretKey))
	h.Write(data)
	sign := base64.URLEncoding.EncodeToString(h.Sum(nil))
	token := fmt.Sprint(qiniuAccessKey, ":", sign)

	req.Header.Set("Authorization", fmt.Sprint("QBox ", token))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	res, err := base.HTTPDo(nil, req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	b, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	if res.StatusCode == http.StatusOK {
		return b, nil
	}

	return nil, fmt.Errorf("GET %s: %s: %s", url, res.Status, b)
}

// thousandsCommaSeperated returns a thousands comma seperated string for the n.
func thousandsCommaSeperated(n int64) string {
	in := strconv.FormatInt(n, 10)
	numOfDigits := len(in)
	if n < 0 {
		numOfDigits--
	}

	numOfCommas := (numOfDigits - 1) / 3

	out := make([]byte, len(in)+numOfCommas)
	if n < 0 {
		in, out[0] = in[1:], '-'
	}

	for i, j, k := len(in)-1, len(out)-1, 0; ; i, j = i-1, j-1 {
		out[j] = in[i]
		if i == 0 {
			return string(out)
		}
		if k++; k == 3 {
			j, k = j-1, 0
			out[j] = ','
		}
	}
}
