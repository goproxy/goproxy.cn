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
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
)

var (
	// goproxyViper is used to get the configuration items of the Goproxy.
	goproxyViper = base.Viper.Sub("goproxy")

	// hhGoproxy is an instance of the `goproxy.Goproxy`.
	hhGoproxy = &goproxy.Goproxy{
		GoBinName:           goproxyViper.GetString("go_bin_name"),
		Cacher:              &goproxyCacher{},
		CacherMaxCacheBytes: goproxyViper.GetInt("cacher_max_cache_bytes"),
		ProxiedSUMDBs:       goproxyViper.GetStringSlice("proxied_sumdbs"),
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

	name, err := url.PathUnescape(req.ParamValue("*").String())
	if err != nil || strings.HasSuffix(name, "/") {
		return CacheableNotFound(req, res, 86400)
	}

	if !goproxyAutoRedirect || path.Ext(name) != ".zip" {
		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	if strings.Contains(name, "..") {
		for _, part := range strings.Split(name, "/") {
			if part == ".." {
				return CacheableNotFound(req, res, 86400)
			}
		}
	}

	name = strings.TrimPrefix(path.Clean(name), "/")
	if !validGoproxyCacheName(name) {
		return CacheableNotFound(req, res, 86400)
	}

	var objectInfo minio.ObjectInfo
	if err := retryQiniuKodoDo(req.Context, func(
		ctx context.Context,
	) (err error) {
		objectInfo, err = qiniuKodoClient.StatObject(
			ctx,
			qiniuKodoBucketName,
			name,
			minio.StatObjectOptions{},
		)
		return err
	}); err != nil {
		if isNotFoundMinIOError(err) {
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

// Get implements the `goproxy.Cacher`.
func (gc *goproxyCacher) Get(
	ctx context.Context,
	name string,
) (io.ReadCloser, error) {
	var (
		object     *minio.Object
		objectInfo minio.ObjectInfo
	)

	if err := retryQiniuKodoDo(ctx, func(ctx context.Context) (err error) {
		object, err = qiniuKodoClient.GetObject(
			ctx,
			qiniuKodoBucketName,
			name,
			minio.GetObjectOptions{},
		)
		if err != nil {
			return err
		}

		objectInfo, err = object.Stat()
		if err != nil {
			object.Close()
		}

		return err
	}); err != nil {
		if isNotFoundMinIOError(err) {
			return nil, fs.ErrNotExist
		}

		return nil, err
	}

	checksum, _ := hex.DecodeString(objectInfo.ETag)
	if len(checksum) != md5.Size {
		eTagChecksum := md5.Sum([]byte(objectInfo.ETag))
		checksum = eTagChecksum[:]
	}

	return &goproxyCacheReader{
		ReadSeekCloser: object,
		modTime:        objectInfo.LastModified,
		checksum:       checksum,
	}, nil
}

// Put implements the `goproxy.Cacher`.
func (gc *goproxyCacher) Put(
	ctx context.Context,
	name string,
	content io.ReadSeeker,
) error {
	if err := retryQiniuKodoDo(ctx, func(ctx context.Context) error {
		_, err := qiniuKodoClient.StatObject(
			ctx,
			qiniuKodoBucketName,
			name,
			minio.StatObjectOptions{},
		)
		return err
	}); err == nil {
		return nil
	} else if !isNotFoundMinIOError(err) {
		return err
	}

	return qiniuKodoUpload(ctx, name, content)
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

// validGoproxyCacheName reports whether the name is a valid Goproxy cache name.
func validGoproxyCacheName(name string) bool {
	nameParts := strings.Split(name, "/@v/")
	if len(nameParts) != 2 {
		return false
	}

	if _, err := module.UnescapePath(nameParts[0]); err != nil {
		return false
	}

	nameBase := path.Base(name)
	nameExt := path.Ext(nameBase)
	switch nameExt {
	case ".info", ".mod", ".zip":
	default:
		return false
	}

	escapedModuleVersion := strings.TrimSuffix(nameBase, nameExt)
	moduleVersion, err := module.UnescapeVersion(escapedModuleVersion)
	if err != nil {
		return false
	}

	return semver.IsValid(moduleVersion)
}
