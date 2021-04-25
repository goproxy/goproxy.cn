package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/minio/minio-go/v7"
)

var (
	// goproxyViper is used to get the configuration items of the Goproxy.
	goproxyViper = base.Viper.Sub("goproxy")

	// hhGoproxy is an instance of the `goproxy.Goproxy`.
	hhGoproxy = &goproxy.Goproxy{
		Cacher: &goproxyCacher{},
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
			DialContext: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,
			MaxIdleConnsPerHost:   200,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			ForceAttemptHTTP2:     true,
		},
		ErrorLogger: log.New(base.Logger, "", 0),
	}

	// goproxyFetchTimeout is the maximum duration allowed for Goproxy to
	// fetch a module.
	goproxyFetchTimeout = goproxyViper.GetDuration("fetch_timeout")

	// goproxyAutoRedirect indicates whether the automatic redirection
	// feature is enabled for Goproxy.
	goproxyAutoRedirect = goproxyViper.GetBool("auto_redirect")

	// goproxyAutoRedirectMinSize is the minimum size of the Goproxy used to
	// limit at least how big Goproxy cache can be automatically redirected.
	goproxyAutoRedirectMinSize = goproxyViper.GetInt64("auto_redirect_min_size")
)

func init() {
	if err := goproxyViper.Unmarshal(hhGoproxy); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to unmarshal goproxy configuration items")
	}

	base.Air.BATCH(getHeadMethods, "/*", hGoproxy)
}

// hGoproxy handles requests to play with Go module proxy.
func hGoproxy(req *air.Request, res *air.Response) error {
	if goproxyFetchTimeout != 0 {
		var cancel context.CancelFunc
		req.Context, cancel = context.WithTimeout(
			req.Context,
			goproxyFetchTimeout,
		)
		defer cancel()
	}

	name := strings.TrimPrefix(path.Clean(req.RawPath()), "/")
	if !goproxyAutoRedirect || !isAutoRedirectableGoproxyCache(name) {
		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	objectInfo, err := qiniuKodoClient.StatObject(
		req.Context,
		qiniuKodoBucketName,
		name,
		minio.StatObjectOptions{},
	)
	if err != nil {
		if isMinIOObjectNotExist(err) {
			hhGoproxy.ServeHTTP(
				res.HTTPResponseWriter(),
				req.HTTPRequest(),
			)
			return nil
		}

		return err
	}

	if objectInfo.Size < goproxyAutoRedirectMinSize {
		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	u, err := qiniuKodoClient.Presign(
		req.Context,
		req.Method,
		qiniuKodoBucketName,
		objectInfo.Key,
		7*24*time.Hour,
		url.Values{
			"response-cache-control": []string{
				"public, max-age=604800",
			},
		},
	)
	if err != nil {
		return err
	}

	return res.Redirect(u.String())
}

// goproxyCacher implements the `goproxy.Cacher`.
type goproxyCacher struct{}

// Cache implements the `goproxy.Cacher`.
func (gc *goproxyCacher) Get(
	ctx context.Context,
	name string,
) (io.ReadCloser, error) {
	objectInfo, err := qiniuKodoClient.StatObject(
		ctx,
		qiniuKodoBucketName,
		name,
		minio.StatObjectOptions{},
	)
	if err != nil {
		if isMinIOObjectNotExist(err) {
			return nil, fs.ErrNotExist
		}

		return nil, err
	}

	checksum, _ := hex.DecodeString(objectInfo.ETag)
	if len(checksum) != md5.Size {
		eTagChecksum := md5.Sum([]byte(objectInfo.ETag))
		checksum = eTagChecksum[:]
	}

	object, err := qiniuKodoClient.GetObject(
		ctx,
		qiniuKodoBucketName,
		objectInfo.Key,
		minio.GetObjectOptions{},
	)
	if err != nil {
		return nil, err
	}

	return &goproxyCacheReader{
		ReadSeekCloser: object,
		modTime:        objectInfo.LastModified,
		checksum:       checksum,
	}, nil
}

// SetCache implements the `goproxy.Cacher`.
func (gc *goproxyCacher) Set(
	ctx context.Context,
	name string,
	content io.ReadSeeker,
) error {
	if _, err := qiniuKodoClient.StatObject(
		base.Context,
		qiniuKodoBucketName,
		name,
		minio.StatObjectOptions{},
	); err == nil {
		return nil
	} else if !isMinIOObjectNotExist(err) {
		return err
	}

	size, err := content.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	} else if _, err := content.Seek(0, io.SeekStart); err != nil {
		return err
	}

	var contentType string
	switch path.Ext(name) {
	case ".info":
		contentType = "application/json; charset=utf-8"
	case ".mod":
		contentType = "text/plain; charset=utf-8"
	case ".zip":
		contentType = "application/zip"
	}

	_, err = qiniuKodoClient.PutObject(
		base.Context,
		qiniuKodoBucketName,
		name,
		content,
		size,
		minio.PutObjectOptions{
			ContentType:      contentType,
			DisableMultipart: size < 256<<20,
		},
	)

	return err
}

// goproxyCacheReader is the reader of the cache unit of the `goproxyCacher`.
type goproxyCacheReader struct {
	io.ReadSeekCloser

	modTime  time.Time
	checksum []byte
}

// ModTime returns the modification time of the gcr.
func (gcr *goproxyCacheReader) ModTime() time.Time {
	return gcr.modTime
}

// Checksum returns the checksum of the gcr.
func (gcr *goproxyCacheReader) Checksum() []byte {
	return gcr.checksum
}

// isAutoRedirectableGoproxyCache reports whether the name refers to an
// auto-redirectable Goproxy cache.
func isAutoRedirectableGoproxyCache(name string) bool {
	return !strings.HasPrefix(name, "sumdb/") &&
		strings.Contains(name, "/@v/") &&
		path.Ext(name) == ".zip"
}
