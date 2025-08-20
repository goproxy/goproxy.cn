package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

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
		Secure: qiniuKodoEndpoint.Scheme == "https",
	}

	if qiniuViper.GetBool("kodo_force_path_style") {
		qiniuKodoClientOptions.BucketLookup = minio.BucketLookupPath
	} else {
		qiniuKodoClientOptions.BucketLookup = minio.BucketLookupDNS
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

// CacheableNotFound returns cacheable not found error.
func CacheableNotFound(req *air.Request, res *air.Response, maxAge int) error {
	res.Header.Set(
		"Cache-Control",
		fmt.Sprintf("public, max-age=%d", maxAge),
	)
	return NotFound(req, res)
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
	return res.Render(map[string]any{
		"IsIndexPage": true,
		"ModuleVersionCount": thousandsCommaSeperated(
			int64(moduleVersionCount),
		),
	}, req.LocalizedString("index.html"), "layouts/default.html")
}

// updateModuleVersionsCount updates the `moduleVersionCount`.
func updateModuleVersionsCount() error {
	var statSummary struct {
		ModuleVersionCount int `json:"module_version_count"`
	}

	if err := retryQiniuKodoDo(base.Context, func(
		ctx context.Context,
	) error {
		object, err := qiniuKodoClient.GetObject(
			ctx,
			qiniuKodoBucketName,
			"stats/summary",
			minio.GetObjectOptions{},
		)
		if err != nil {
			return err
		}
		defer object.Close()

		return json.NewDecoder(object).Decode(&statSummary)
	}); err != nil {
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
) (err error) {
	var contentType string
	switch path.Base(name) {
	case "@latest":
		contentType = "application/json; charset=utf-8"
	case "list":
		contentType = "text/plain; charset=utf-8"
	default:
		switch path.Ext(name) {
		case ".info":
			contentType = "application/json; charset=utf-8"
		case ".mod":
			contentType = "text/plain; charset=utf-8"
		case ".zip":
			contentType = "application/zip"
		}
	}

	var size int64
	if f, ok := content.(*os.File); ok {
		fi, err := f.Stat()
		if err != nil {
			return err
		}

		size = fi.Size()
	} else if size, err = content.Seek(0, io.SeekEnd); err != nil {
		return err
	} else if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	if size <= qiniuKodoMultipartUploadPartSize {
		content := content
		if ra, ok := content.(io.ReaderAt); ok {
			content = io.NewSectionReader(ra, 0, size)
		} else if _, err := content.Seek(0, io.SeekStart); err != nil {
			return err
		}

		return retryQiniuKodoDo(ctx, func(ctx context.Context) error {
			_, err := qiniuKodoCore.PutObject(
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
			)
			return err
		})
	}

	var uploadID string
	if err := retryQiniuKodoDo(ctx, func(ctx context.Context) (err error) {
		uploadID, err = qiniuKodoCore.NewMultipartUpload(
			ctx,
			qiniuKodoBucketName,
			name,
			minio.PutObjectOptions{
				ContentType: contentType,
			},
		)
		return err
	}); err != nil {
		return err
	}
	defer func() {
		if err != nil {
			retryQiniuKodoDo(ctx, func(ctx context.Context) error {
				return qiniuKodoCore.AbortMultipartUpload(
					ctx,
					qiniuKodoBucketName,
					name,
					uploadID,
				)
			})
		}
	}()

	var completeParts []minio.CompletePart
	for offset := int64(0); offset < size; {
		partSize := min(qiniuKodoMultipartUploadPartSize, size-offset)

		var part minio.ObjectPart
		if err := retryQiniuKodoDo(ctx, func(
			ctx context.Context,
		) (err error) {
			content := content
			if ra, ok := content.(io.ReaderAt); ok {
				content = io.NewSectionReader(
					ra,
					offset,
					partSize,
				)
			} else if _, err := content.Seek(
				offset,
				io.SeekStart,
			); err != nil {
				return err
			}

			part, err = qiniuKodoCore.PutObjectPart(
				ctx,
				qiniuKodoBucketName,
				name,
				uploadID,
				len(completeParts)+1,
				io.LimitReader(content, partSize),
				partSize,
				minio.PutObjectPartOptions{},
			)

			return err
		}); err != nil {
			return err
		}

		completeParts = append(completeParts, minio.CompletePart{
			PartNumber: part.PartNumber,
			ETag:       part.ETag,
		})

		offset += part.Size
	}

	return retryQiniuKodoDo(ctx, func(ctx context.Context) error {
		_, err := qiniuKodoCore.CompleteMultipartUpload(
			ctx,
			qiniuKodoBucketName,
			name,
			uploadID,
			completeParts,
			minio.PutObjectOptions{
				ContentType: contentType,
			},
		)
		return err
	})
}

// retryQiniuKodoDo retries a Qiniu Cloud Kodo operation in case of some special
// errors.
func retryQiniuKodoDo(
	ctx context.Context,
	f func(ctx context.Context) error,
) error {
	return base.RetryN(ctx, f, func(err error) bool {
		switch minio.ToErrorResponse(err).StatusCode {
		case 573, 579, 599:
			return true
		}

		return false
	}, 100*time.Millisecond, 10)
}

// isNotFoundMinIOError reports whether the err is MinIO not found error.
func isNotFoundMinIOError(err error) bool {
	return minio.ToErrorResponse(err).StatusCode == http.StatusNotFound
}

// thousandsCommaSeperated returns a thousands comma separated string for the n.
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
