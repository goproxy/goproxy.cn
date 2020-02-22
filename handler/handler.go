package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/goproxy/goproxy/cacher"
)

var (
	// qiniuViper is used to get the configuration items of the Qiniu Cloud.
	qiniuViper = base.Viper.Sub("qiniu")

	// goproxyViper is used to get the configuration items of the Goproxy.
	goproxyViper = base.Viper.Sub("goproxy")

	// getHeadMethods is an array contains the GET and the HEAD methods.
	getHeadMethods = []string{http.MethodGet, http.MethodHead}

	// cachemanGas is used to manage the Cache-Control header.
	cachemanGas = cacheman.Gas(cacheman.GasConfig{
		Public:  true,
		MaxAge:  3600,
		SMaxAge: -1,
	})

	// kodoCacher is the `cacher.Kodo` for the Qiniu Cloud Kodo.
	kodoCacher = &cacher.Kodo{
		Endpoint:   qiniuViper.GetString("kodo_endpoint"),
		AccessKey:  qiniuViper.GetString("access_key"),
		SecretKey:  qiniuViper.GetString("secret_key"),
		BucketName: qiniuViper.GetString("kodo_bucket_name"),
	}

	// g is an instance of the `goproxy.Goproxy`.
	g = goproxy.New()
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	base.Air.AddShutdownJob(cancel)

	if err := goproxyViper.UnmarshalKey("goproxy", g); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to unmarshal goproxy configuration items")
	}

	g.GoBinName = goproxyViper.GetString("go_bin_name")
	g.MaxGoBinWorkers = goproxyViper.GetInt("max_go_bin_workers")
	g.Cacher = &localCacher{
		Cacher:         kodoCacher,
		localCacheRoot: goproxyViper.GetString("local_cache_root"),
		settingContext: ctx,
	}

	g.ProxiedSUMDBNames = []string{"sum.golang.org"}
	g.ErrorLogger = log.New(base.Logger, "", 0)
	g.DisableNotFoundLog = true

	base.Air.FILE("/robots.txt", "robots.txt")
	base.Air.FILE("/favicon.ico", "favicon.ico", cachemanGas)
	base.Air.FILE("/apple-touch-icon.png", "apple-touch-icon.png", cachemanGas)
	base.Air.FILES("/assets", base.Air.CofferAssetRoot, cachemanGas)
	base.Air.BATCH(getHeadMethods, "/", indexPage)
	base.Air.BATCH(getHeadMethods, "/faq", faqPage)
	base.Air.BATCH(nil, "/*", proxy)
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

// indexPage handles requests to get index page.
func indexPage(req *air.Request, res *air.Response) error {
	const indexPageURLBase = "https://github.com/goproxy/goproxy.cn" +
		"/blob/master/README.md"
	return res.WriteHTML(fmt.Sprintf(
		"<meta http-equiv=refresh content=0;url=%s>",
		req.LocalizedString(indexPageURLBase),
	))
}

// faqPage handles requests to get FAQ page.
func faqPage(req *air.Request, res *air.Response) error {
	const faqPageURLBase = "https://github.com/goproxy/goproxy.cn/wiki/FAQ"
	return res.WriteHTML(fmt.Sprintf(
		"<meta http-equiv=refresh content=0;url=%s>",
		req.LocalizedString(faqPageURLBase),
	))
}

// proxy handles requests to play with Go module proxy.
func proxy(req *air.Request, res *air.Response) error {
	name := strings.TrimLeft(path.Clean(req.RawPath()), "/")
	if isModuleCacheFile(name) && goproxyViper.GetBool("auto_redirect") {
		if _, err := kodoCacher.Cache(req.Context, name); err == nil {
			u := fmt.Sprintf(
				"%s/%s?e=%d",
				qiniuViper.GetString("kodo_bucket_endpoint"),
				name,
				time.Now().Add(time.Hour).Unix(),
			)
			h := hmac.New(
				sha1.New,
				[]byte(qiniuViper.GetString("secret_key")),
			)
			h.Write([]byte(u))
			u = fmt.Sprintf(
				"%s&token=%s:%s",
				u,
				qiniuViper.GetString("access_key"),
				base64.URLEncoding.EncodeToString(h.Sum(nil)),
			)

			return res.Redirect(u)
		} else if !errors.Is(err, goproxy.ErrCacheNotFound) {
			return err
		}
	}

	g.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

	return nil
}

// localCacher implements the `goproxy.Cacher`.
type localCacher struct {
	goproxy.Cacher

	localCacheRoot    string
	settingContext    context.Context
	settingCaches     sync.Map
	startSetCacheOnce sync.Once
}

// startSetCache starts the cache setting of the lc.
func (lc *localCacher) startSetCache() {
	go func() {
		for {
			time.Sleep(time.Second)
			if lc.settingContext.Err() != nil {
				return
			}

			lc.settingCaches.Range(func(k, v interface{}) bool {
				if lc.settingContext.Err() != nil {
					return false
				}

				localCacheFile, err := os.Open(k.(string))
				if err != nil {
					if os.IsNotExist(err) {
						lc.settingCaches.Delete(k)
					}

					return false
				}
				defer os.Remove(localCacheFile.Name())

				lc.settingCaches.Delete(k)

				cache := v.(goproxy.Cache)
				lc.Cacher.SetCache(
					lc.settingContext,
					&localCache{
						File:     localCacheFile,
						name:     cache.Name(),
						mimeType: cache.MIMEType(),
						size:     cache.Size(),
						modTime:  cache.ModTime(),
						checksum: cache.Checksum(),
					},
				)

				return true
			})
		}
	}()
}

// SetCache implements the `goproxy.Cacher`.
func (lc *localCacher) SetCache(ctx context.Context, c goproxy.Cache) error {
	lc.startSetCacheOnce.Do(lc.startSetCache)

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

	lc.settingCaches.Store(localCacheFile.Name(), c)

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
