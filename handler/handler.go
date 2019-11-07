package handler

import (
	"context"
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
	"github.com/qiniu/api.v7/v7/auth/qbox"
	"github.com/qiniu/api.v7/v7/storage"
	"github.com/rs/zerolog/log"
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

	// qiniuMac is the credentials of the Qiniu Cloud.
	qiniuMac *qbox.Mac

	// qiniuKodoBucketManager is the manager of the Qiniu Cloud Kodo.
	qiniuKodoBucketManager *storage.BucketManager

	// g is an instance of the `goproxy.Goproxy`.
	g = goproxy.New()
)

func init() {
	qiniuMac = qbox.NewMac(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)

	qiniuKodoRegion, err := storage.GetRegion(
		cfg.Qiniu.AccessKey,
		cfg.Qiniu.KodoBucketName,
	)
	if err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to get qiniu cloud kodo region")
	}

	qiniuKodoConfig := &storage.Config{
		Region:        qiniuKodoRegion,
		UseHTTPS:      true,
		UseCdnDomains: true,
	}

	qiniuKodoBucketManager = storage.NewBucketManager(
		qiniuMac,
		qiniuKodoConfig,
	)

	g.GoBinName = cfg.Goproxy.GoBinName
	g.MaxGoBinWorkers = cfg.Goproxy.MaxGoBinWorkers
	g.Cacher = &localCacher{
		Cacher: &cacher.Kodo{
			Endpoint:   cfg.Qiniu.KodoEndpoint,
			AccessKey:  cfg.Qiniu.AccessKey,
			SecretKey:  cfg.Qiniu.SecretKey,
			BucketName: cfg.Qiniu.KodoBucketName,
		},
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
	const indexPageURLBase = "https://github.com/goproxy/goproxy.cn" +
		"/blob/master/README.md"
	return res.Redirect(req.LocalizedString(indexPageURLBase))
}

// proxy handles requests to play with Go module proxy.
func proxy(req *air.Request, res *air.Response) error {
	if p, _ := splitPathQuery(req.Path); path.Ext(p) == ".zip" {
		fn := strings.TrimLeft(path.Clean(p), "/")
		if _, err := qiniuKodoBucketManager.Stat(
			cfg.Qiniu.KodoBucketName,
			fn,
		); err == nil {
			return res.Redirect(storage.MakePrivateURL(
				qiniuMac,
				cfg.Qiniu.KodoBucketEndpoint,
				fn,
				time.Now().Add(time.Hour).Unix(),
			))
		} else if !isKodoFileNotExist(err) {
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

// isKodoFileNotExist reports whether the err means a Qiniu Cloud Kodo file is
// not exist.
func isKodoFileNotExist(err error) bool {
	return err != nil && err.Error() == "no such file or directory"
}
