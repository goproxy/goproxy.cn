package handler

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"strings"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/goproxy/goproxy/cacher"
	"github.com/qiniu/api.v7/v7/auth"
)

var (
	// a is the `air.Default`.
	a = air.Default

	// getHeadMethods is an array contains the GET and the HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// cachemanGas is used to manage the Cache-Control header.
	cachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})

	// qiniuCredentials is the `auth.Credentials` for the Qiniu Cloud.
	qiniuCredentials *auth.Credentials

	// kodoCacher is the implementation of the `goproxy.Cacher` for the
	// Qiniu Cloud Kodo.
	kodoCacher *cacher.Kodo

	// g is an instance of the `goproxy.Goproxy`.
	g = goproxy.New()
)

func init() {
	qiniuCredentials = auth.New(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)

	kodoCacher = &cacher.Kodo{
		Endpoint:   cfg.Qiniu.KodoEndpoint,
		AccessKey:  cfg.Qiniu.AccessKey,
		SecretKey:  cfg.Qiniu.SecretKey,
		BucketName: cfg.Qiniu.KodoBucketName,
	}

	g.GoBinName = cfg.Goproxy.GoBinName
	g.MaxGoBinWorkers = cfg.Goproxy.MaxGoBinWorkers
	g.Cacher = &localCacher{
		Cacher:         kodoCacher,
		localCacheRoot: cfg.Goproxy.LocalCacheRoot,
	}

	g.MaxZIPCacheBytes = cfg.Goproxy.MaxZIPCacheBytes
	g.ErrorLogger = a.ErrorLogger
	g.DisableNotFoundLog = true

	a.FILE("/robots.txt", "robots.txt")
	a.FILE("/favicon.ico", "favicon.ico", cachemanGas)
	a.FILE("/apple-touch-icon.png", "apple-touch-icon.png", cachemanGas)
	a.FILES("/assets", a.CofferAssetRoot, cachemanGas)
	a.BATCH(getHeadMethods, "/", indexPage, cachemanGas)
	a.BATCH(nil, "/*", proxy)
}

// indexPage handles requests to get index page.
func indexPage(req *air.Request, res *air.Response) error {
	const indexHTML = "<meta http-equiv=refresh " +
		"content=0;url=https://github.com/goproxy/goproxy.cn>"
	return res.WriteHTML(indexHTML)
}

// proxy handles requests to play with Go module proxy.
func proxy(req *air.Request, res *air.Response) error {
	name, _ := splitPathQuery(req.Path)
	name = path.Clean(name)
	name = strings.TrimPrefix(name, g.PathPrefix)
	name = strings.TrimLeft(name, "/")
	if cfg.Goproxy.AutoRedirection && isModuleCacheFile(name) {
		if _, err := kodoCacher.Cache(req.Context, name); err == nil {
			u := fmt.Sprintf(
				"%s/%s?e=%d",
				cfg.Qiniu.KodoBucketEndpoint,
				name,
				time.Now().Add(time.Hour).Unix(),
			)

			token := qiniuCredentials.Sign([]byte(u))

			return res.Redirect(fmt.Sprint(u, "&token=", token))
		} else if err != goproxy.ErrCacheNotFound {
			return err
		}
	}

	g.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

	return nil
}

// localCacher implements the `goproxy.Cacher`.
type localCacher struct {
	goproxy.Cacher

	localCacheRoot string
}

// SetCache implements the `goproxy.Cacher`.
func (lc *localCacher) SetCache(ctx context.Context, c goproxy.Cache) error {
	localCacheFile, err := ioutil.TempFile(lc.localCacheRoot, "")
	if err != nil {
		return err
	}

	hijackedLocalCacheRemoval := false
	defer func() {
		if !hijackedLocalCacheRemoval {
			os.Remove(localCacheFile.Name())
		}
	}()

	if _, err := io.Copy(localCacheFile, c); err != nil {
		return err
	}

	if err := localCacheFile.Close(); err != nil {
		return err
	}

	hijackedLocalCacheRemoval = true
	go func() {
		defer os.Remove(localCacheFile.Name())

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Minute,
		)
		defer cancel()

		localCacheFile, err := os.Open(localCacheFile.Name())
		if err != nil {
			return
		}
		defer localCacheFile.Close()

		lc.Cacher.SetCache(ctx, &localCache{
			File:     localCacheFile,
			name:     c.Name(),
			mimeType: c.MIMEType(),
			size:     c.Size(),
			modTime:  c.ModTime(),
			checksum: c.Checksum(),
		})
	}()

	return nil
}

// localCache implements the `goproxy.Cache`. It is the cache unit of the
// `localCacher`.
type localCache struct {
	*os.File

	name     string
	mimeType string
	size     int64
	modTime  time.Time
	checksum []byte
}

// Name implements the `goproxy.Cache`.
func (lc *localCache) Name() string {
	return lc.name
}

// MIMEType implements the `goproxy.Cache`.
func (lc *localCache) MIMEType() string {
	return lc.mimeType
}

// Size implements the `goproxy.Cache`.
func (lc *localCache) Size() int64 {
	return lc.size
}

// ModTime implements the `goproxy.Cache`.
func (lc *localCache) ModTime() time.Time {
	return lc.modTime
}

// Checksum implements the `goproxy.Cache`.
func (lc *localCache) Checksum() []byte {
	return lc.checksum
}

// splitPathQuery splits the p of the form "path?query" into path and query.
func splitPathQuery(p string) (path, query string) {
	i, l := 0, len(p)
	for ; i < l && p[i] != '?'; i++ {
	}

	if i < l {
		return p[:i], p[i+1:]
	}

	return p, ""
}

// isModuleCacheFile reports whether the named file is a module cache.
func isModuleCacheFile(name string) bool {
	if !strings.Contains(name, "/@v/v") {
		return false
	}

	switch path.Ext(name) {
	case ".info", ".mod", ".zip":
		return true
	}

	return false
}
