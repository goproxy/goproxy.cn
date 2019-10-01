package handler

import (
	"context"
	"hash"
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
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"github.com/rs/zerolog/log"
)

var (
	// a is the `air.Default`.
	a = air.Default

	// g is an instance of the `goproxy.Goproxy`.
	g = goproxy.New()

	// kodoMac is the credentials of the Qiniu Cloud Kodo.
	kodoMac *qbox.Mac

	// kodoBucketManager is the manager of the Qiniu Cloud Kodo.
	kodoBucketManager *storage.BucketManager

	// getHeadMethods is an array contains the GET and the HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// cachemanGas is used to manage the Cache-Control header.
	cachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})
)

func init() {
	g.GoBinName = cfg.Goproxy.GoBinName
	g.MaxGoBinWorkers = cfg.Goproxy.MaxGoBinWorkers
	g.Cacher = &kodoCacher{
		kodoCacher: &cacher.Kodo{
			Endpoint:   cfg.Kodo.Endpoint,
			AccessKey:  cfg.Kodo.AccessKey,
			SecretKey:  cfg.Kodo.SecretKey,
			BucketName: cfg.Kodo.BucketName,
		},
	}

	g.MaxZIPCacheBytes = cfg.Goproxy.MaxZIPCacheBytes
	g.ErrorLogger = a.ErrorLogger
	g.DisableNotFoundLog = true

	kodoMac = qbox.NewMac(cfg.Kodo.AccessKey, cfg.Kodo.SecretKey)

	kodoRegion, err := storage.GetRegion(
		cfg.Kodo.AccessKey,
		cfg.Kodo.BucketName,
	)
	if err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to get qiniu cloud kodo region")
	}

	kodoBucketManager = storage.NewBucketManager(kodoMac, &storage.Config{
		Region: kodoRegion,
	})

	a.FILE("/robots.txt", "robots.txt")
	a.FILE("/favicon.ico", "favicon.ico", cachemanGas)
	a.FILE("/apple-touch-icon.png", "apple-touch-icon.png", cachemanGas)
	a.FILES("/assets", a.CofferAssetRoot, cachemanGas)
	a.BATCH(getHeadMethods, "/", indexPageHandler, cachemanGas)
	a.BATCH(nil, "/*", goproxyHandler)
}

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	return res.Redirect("https://github.com/goproxy/goproxy.cn")
}

// goproxyHandler handles requests to play with Go module proxy.
func goproxyHandler(req *air.Request, res *air.Response) error {
	if p, _ := splitPathQuery(req.Path); path.Ext(p) == ".zip" {
		fk := strings.TrimLeft(path.Clean(p), "/")
		fi, err := kodoBucketManager.Stat(cfg.Kodo.BucketName, fk)
		if err != nil {
			if !isKodoFileNotExist(err) {
				return err
			}
		} else if fi.Fsize > 10<<20 { // File size > 10 MB
			fu := storage.MakePrivateURL(
				kodoMac,
				cfg.Kodo.BucketEndpoint,
				fk,
				time.Now().Add(time.Hour).Unix(),
			)
			return res.Redirect(fu)
		}
	}

	g.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

	return nil
}

// kodoCacher implements the `goproxy.Cacher`.
type kodoCacher struct {
	// kodoCacher is the underlying cacher.
	kodoCacher *cacher.Kodo
}

// NewHash implements the `goproxy.Cacher`.
func (kc *kodoCacher) NewHash() hash.Hash {
	return kc.kodoCacher.NewHash()
}

// Cache implements the `goproxy.Cacher`.
func (kc *kodoCacher) Cache(
	ctx context.Context,
	name string,
) (goproxy.Cache, error) {
	return kc.kodoCacher.Cache(ctx, name)
}

// SetCache implements the `goproxy.Cacher`.
func (kc *kodoCacher) SetCache(ctx context.Context, c goproxy.Cache) error {
	localCache, err := ioutil.TempFile(cfg.Goproxy.LocalCacheRoot, "")
	if err != nil {
		return err
	}

	hijackedLocalCacheRemoval := false
	defer func() {
		if !hijackedLocalCacheRemoval {
			os.Remove(localCache.Name())
		}
	}()

	if _, err := io.Copy(localCache, c); err != nil {
		return err
	}

	if err := localCache.Close(); err != nil {
		return err
	}

	hijackedLocalCacheRemoval = true
	go func() {
		defer os.Remove(localCache.Name())

		ctx, cancel := context.WithTimeout(
			context.Background(),
			10*time.Minute,
		)
		defer cancel()

		lc, err := os.Open(localCache.Name())
		if err != nil {
			return
		}

		dc := &diskCache{
			file:     lc,
			name:     c.Name(),
			mimeType: c.MIMEType(),
			size:     c.Size(),
			modTime:  c.ModTime(),
			checksum: c.Checksum(),
		}
		defer dc.Close()

		kc.kodoCacher.SetCache(ctx, dc)
	}()

	return nil
}

// diskCache implements the `goproxy.Cache`.
type diskCache struct {
	file     *os.File
	name     string
	mimeType string
	size     int64
	modTime  time.Time
	checksum []byte
}

// Read implements the `goproxy.Cache`.
func (dc *diskCache) Read(b []byte) (int, error) {
	return dc.file.Read(b)
}

// Seek implements the `goproxy.Cache`.
func (dc *diskCache) Seek(offset int64, whence int) (int64, error) {
	return dc.file.Seek(offset, whence)
}

// Close implements the `goproxy.Cache`.
func (dc *diskCache) Close() error {
	return dc.file.Close()
}

// Name implements the `goproxy.Cache`.
func (dc *diskCache) Name() string {
	return dc.name
}

// MIMEType implements the `goproxy.Cache`.
func (dc *diskCache) MIMEType() string {
	return dc.mimeType
}

// Size implements the `goproxy.Cache`.
func (dc *diskCache) Size() int64 {
	return dc.size
}

// ModTime implements the `goproxy.Cache`.
func (dc *diskCache) ModTime() time.Time {
	return dc.modTime
}

// Checksum implements the `goproxy.Cache`.
func (dc *diskCache) Checksum() []byte {
	return dc.checksum
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

// isKodoFileNotExist reports whether the err means a Kodo file is not exist.
func isKodoFileNotExist(err error) bool {
	return err != nil && err.Error() == "no such file or directory"
}
