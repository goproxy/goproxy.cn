package handler

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/robfig/cron/v3"
)

var (
	// qiniuViper is used to get the configuration items of the Qiniu Cloud.
	qiniuViper = base.Viper.Sub("qiniu")

	// qiniuKodoBucketName is the bucket name for the Qiniu Cloud Kodo.
	qiniuKodoBucketName = qiniuViper.GetString("kodo_bucket_name")

	// qiniuKodoMultipartUploadPartSize is the multipart upload part size
	// for the Qiniu Cloud Kodo.
	qiniuKodoMultipartUploadPartSize = qiniuViper.GetInt64("kodo_multipart_upload_part_size")

	// qiniuKodoClient is the client for the Qiniu Cloud Kodo.
	qiniuKodoClient *minio.Client

	// qiniuKodoCore is the core for the Qiniu Cloud Kodo.
	qiniuKodoCore *minio.Core

	// getHeadMethods is an array contains the GET and HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// minutelyCachemanGas is used to manage the Cache-Control header.
	minutelyCachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  60,
		SMaxAge: -1,
	})

	// hourlyCachemanGas is used to manage the Cache-Control header.
	hourlyCachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})

	// moduleVersionCount is the module version count.
	moduleVersionCount int
)

func init() {
	qiniuKodoEndpoint, err := url.Parse(
		qiniuViper.GetString("kodo_endpoint"),
	)
	if err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to parse qiniu kodo endpoint")
	}

	qiniuKodoClientOptions := &minio.Options{
		Creds: credentials.NewStaticV4(
			qiniuViper.GetString("access_key"),
			qiniuViper.GetString("secret_key"),
			"",
		),
		Secure:       qiniuKodoEndpoint.Scheme == "https",
		BucketLookup: minio.BucketLookupPath,
	}

	qiniuKodoEndpoint.Scheme = ""
	qiniuKodoClient, err = minio.New(
		strings.TrimPrefix(qiniuKodoEndpoint.String(), "//"),
		qiniuKodoClientOptions,
	)
	if err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to create qiniu kodo client")
	}

	qiniuKodoCore = &minio.Core{
		Client: qiniuKodoClient,
	}

	if err := updateModuleVersionsCount(); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to initialize module version count")
	}

	if _, err := base.Cron.AddJob(
		"* * * * *", // Every minute
		cron.NewChain(
			cron.SkipIfStillRunning(cron.DiscardLogger),
		).Then(cron.FuncJob(func() {
			err := updateModuleVersionsCount()
			if err == nil {
				return
			}

			base.Logger.Error().Err(err).
				Msg("failed to update module version count")
		})),
	); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to add module version count update cron " +
				"job")
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

// NotFound returns not found error.
func NotFound(req *air.Request, res *air.Response) error {
	res.Status = http.StatusNotFound
	return errors.New(strings.ToLower(http.StatusText(res.Status)))
}

// MethodNotAllowed returns method not allowed error.
func MethodNotAllowed(req *air.Request, res *air.Response) error {
	res.Status = http.StatusMethodNotAllowed
	return errors.New(strings.ToLower(http.StatusText(res.Status)))
}

// Error handles errors.
func Error(err error, req *air.Request, res *air.Response) {
	if res.Written {
		return
	}

	if !req.Air.DebugMode && res.Status == http.StatusInternalServerError {
		res.WriteString(strings.ToLower(http.StatusText(res.Status)))
	} else {
		res.WriteString(err.Error())
	}
}

// hIndexPage handles requests to get index page.
func hIndexPage(req *air.Request, res *air.Response) error {
	return res.Render(map[string]interface{}{
		"IsIndexPage": true,
		"ModuleVersionCount": thousandsCommaSeperated(
			int64(moduleVersionCount),
		),
	}, req.LocalizedString("index.html"), "layouts/default.html")
}

// updateModuleVersionsCount updates the `moduleVersionCount`.
func updateModuleVersionsCount() error {
	object, err := qiniuKodoClient.GetObject(
		base.Context,
		qiniuKodoBucketName,
		"stats/summary",
		minio.GetObjectOptions{},
	)
	if err != nil {
		return err
	}
	defer object.Close()

	b, err := io.ReadAll(object)
	if err != nil {
		return err
	}

	var statSummary struct {
		ModuleVersionCount int `json:"module_version_count"`
	}

	if err := json.Unmarshal(b, &statSummary); err != nil {
		return err
	}

	moduleVersionCount = statSummary.ModuleVersionCount

	return nil
}

// qiniuKodoUpload uploads the content with the name to the Qiniu Cloud Kodo.
func qiniuKodoUpload(
	ctx context.Context,
	name string,
	content io.ReadSeeker,
) error {
	var contentType string
	switch path.Ext(name) {
	case ".info":
		contentType = "application/json; charset=utf-8"
	case ".mod":
		contentType = "text/plain; charset=utf-8"
	case ".zip":
		contentType = "application/zip"
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}

	if size > qiniuKodoMultipartUploadPartSize {
		if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		return qiniuKodoMultipartUpload(
			ctx,
			name,
			content,
			contentType,
			size,
			qiniuKodoMultipartUploadPartSize,
		)
	}

PutObject:
	if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if _, err := qiniuKodoCore.PutObject(
		ctx,
		qiniuKodoBucketName,
		name,
		content,
		size,
		"",
		"",
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	); err != nil {
		if isRetryableMinIOError(err) {
			goto PutObject
		}

		return err
	}

	return nil
}

// qiniuKodoMultipartUpload is similar to the `qiniuKodoUpload`, but uses
// multiple uploads.
func qiniuKodoMultipartUpload(
	ctx context.Context,
	name string,
	content io.ReadSeeker,
	contentType string,
	size int64,
	partSize int64,
) (err error) {
	var uploadID string
	defer func() {
		if err != nil && uploadID != "" {
			qiniuKodoCore.AbortMultipartUpload(
				ctx,
				qiniuKodoBucketName,
				name,
				uploadID,
			)
		}
	}()

NewMultipartUpload:
	if uploadID, err = qiniuKodoCore.NewMultipartUpload(
		ctx,
		qiniuKodoBucketName,
		name,
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	); err != nil {
		if isRetryableMinIOError(err) {
			goto NewMultipartUpload
		}

		return err
	}

	var completeParts []minio.CompletePart
	for offset := int64(0); offset < size; offset += partSize {
		partSize := partSize
		if r := size - offset; r < partSize {
			partSize = r
		}

	PutObjectPart:
		if _, err := content.Seek(offset, io.SeekStart); err != nil {
			return err
		}

		part, err := qiniuKodoCore.PutObjectPart(
			ctx,
			qiniuKodoBucketName,
			name,
			uploadID,
			len(completeParts)+1,
			io.LimitReader(content, partSize),
			partSize,
			"",
			"",
			nil,
		)
		if err != nil {
			if isRetryableMinIOError(err) {
				goto PutObjectPart
			}

			return err
		}

		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})
	}

CompleteMultipartUpload:
	if _, err := qiniuKodoCore.CompleteMultipartUpload(
		ctx,
		qiniuKodoBucketName,
		name,
		uploadID,
		completeParts,
	); err != nil {
		if isRetryableMinIOError(err) {
			goto CompleteMultipartUpload
		}

		return err
	}

	return nil
}

// isRetryableMinIOError reports whether the err is a retryable MinIO error.
func isRetryableMinIOError(err error) bool {
	switch minio.ToErrorResponse(err).StatusCode {
	case http.StatusTooManyRequests,
		http.StatusInternalServerError,
		http.StatusBadGateway,
		http.StatusServiceUnavailable,
		http.StatusGatewayTimeout:
		return true
	}

	t, ok := err.(interface{ Timeout() bool })

	return ok && t.Timeout()
}

// isNotFoundMinIOError reports whether the err is MinIO not found error.
func isNotFoundMinIOError(err error) bool {
	return minio.ToErrorResponse(err).StatusCode == http.StatusNotFound
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
