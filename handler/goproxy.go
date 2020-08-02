package handler

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
)

var (
	// goproxyViper is used to get the configuration items of the Goproxy.
	goproxyViper = base.Viper.Sub("goproxy")

	// hhGoproxy is an instance of the `goproxy.Goproxy`.
	hhGoproxy = goproxy.New()

	// goproxyAutoRedirect indicates whether the automatic redirection
	// feature is enabled for Goproxy.
	goproxyAutoRedirect = goproxyViper.GetBool("auto_redirect")

	// goproxyAutoRedirectMinSize is the minimum size of the Goproxy used to
	// limit at least how big Goproxy cache can be automatically redirected.
	goproxyAutoRedirectMinSize = goproxyViper.GetInt64("auto_redirect_min_size")

	// qiniuKodoBucketEndpoint is the bucket endpoint for the Qiniu Cloud
	// Kodo.
	qiniuKodoBucketEndpoint = qiniuViper.GetString("kodo_bucket_endpoint")
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	base.Air.AddShutdownJob(cancel)

	if err := goproxyViper.Unmarshal(hhGoproxy); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to unmarshal goproxy configuration items")
	}

	goproxyLocalCacheRoot, err := ioutil.TempDir(
		goproxyViper.GetString("local_cache_root"),
		"goproxy-china-local-caches",
	)
	if err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to create goproxy local cache root")
	}
	base.Air.AddShutdownJob(func() {
		for i := 0; i < 60; i++ {
			time.Sleep(time.Second)
			err := os.RemoveAll(goproxyLocalCacheRoot)
			if err == nil {
				break
			}
		}
	})

	hhGoproxy.Cacher = &goproxyCacher{
		Cacher:         qiniuKodoCacher,
		localCacheRoot: goproxyLocalCacheRoot,
		settingContext: ctx,
	}

	hhGoproxy.ErrorLogger = log.New(base.Logger, "", 0)

	base.Air.BATCH(nil, "/*", hGoproxy)
}

// hGoproxy handles requests to play with Go module proxy.
func hGoproxy(req *air.Request, res *air.Response) error {
	name := strings.TrimPrefix(path.Clean(req.RawPath()), "/")
	if !goproxyAutoRedirect || !isAutoRedirectableGoproxyCache(name) {
		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	cache, err := hhGoproxy.Cacher.Cache(req.Context, name)
	if err != nil {
		if !errors.Is(err, goproxy.ErrCacheNotFound) {
			return err
		}

		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

		return nil
	}
	defer cache.Close()

	if cache.Size() < goproxyAutoRedirectMinSize {
		hhGoproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	e := time.Now().Add(24 * time.Hour).Unix()
	u := fmt.Sprintf("%s/%s?e=%d", qiniuKodoBucketEndpoint, name, e)
	h := hmac.New(sha1.New, []byte(qiniuSecretKey))
	h.Write([]byte(u))
	s := base64.URLEncoding.EncodeToString(h.Sum(nil))
	u = fmt.Sprintf("%s&token=%s:%s", u, qiniuAccessKey, s)

	return res.Redirect(u)
}

// goproxyCacher implements the `goproxy.Cacher`.
type goproxyCacher struct {
	goproxy.Cacher

	localCacheRoot    string
	settingContext    context.Context
	settingMutex      sync.Mutex
	settingCaches     sync.Map
	startSetCacheOnce sync.Once
}

// startSetCache starts the cache setting of the gc.
func (gc *goproxyCacher) startSetCache() {
	go func() {
		for {
			time.Sleep(time.Second)
			if gc.settingContext.Err() != nil {
				return
			}

			gc.settingCaches.Range(func(k, v interface{}) bool {
				if gc.settingContext.Err() != nil {
					return false
				}

				localCacheFile, err := os.Open(k.(string))
				if err != nil {
					if os.IsNotExist(err) {
						gc.settingCaches.Delete(k)
					}

					return true
				}
				defer func() {
					gc.settingMutex.Lock()
					os.Remove(localCacheFile.Name())
					gc.settingMutex.Unlock()
				}()

				gc.settingCaches.Delete(k)

				cache := v.(goproxy.Cache)
				gc.Cacher.SetCache(
					gc.settingContext,
					&goproxyCache{
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
func (gc *goproxyCacher) SetCache(ctx context.Context, c goproxy.Cache) error {
	gc.startSetCacheOnce.Do(gc.startSetCache)

	cacheNameChecksum := sha256.Sum256([]byte(c.Name()))

	localCacheFileName := filepath.Join(
		gc.localCacheRoot,
		hex.EncodeToString(cacheNameChecksum[:]),
	)

	gc.settingMutex.Lock()

	if _, err := os.Stat(localCacheFileName); err == nil {
		gc.settingMutex.Unlock()
		return nil
	} else if !os.IsNotExist(err) {
		gc.settingMutex.Unlock()
		return err
	}

	localCacheFile, err := os.Create(localCacheFileName)
	if err != nil {
		gc.settingMutex.Unlock()
		return err
	}

	gc.settingMutex.Unlock()

	if _, err := io.Copy(localCacheFile, c); err != nil {
		os.Remove(localCacheFile.Name())
		return err
	}

	if err := localCacheFile.Close(); err != nil {
		os.Remove(localCacheFile.Name())
		return err
	}

	gc.settingCaches.Store(localCacheFile.Name(), c)

	return nil
}

// goproxyCache implements the `goproxy.Cache`. It is the cache unit of the
// `goproxyCacher`.
type goproxyCache struct {
	*os.File

	name     string
	mimeType string
	size     int64
	modTime  time.Time
	checksum []byte
}

// Name implements the `goproxy.Cache`.
func (gc *goproxyCache) Name() string {
	return gc.name
}

// MIMEType implements the `goproxy.Cache`.
func (gc *goproxyCache) MIMEType() string {
	return gc.mimeType
}

// Size implements the `goproxy.Cache`.
func (gc *goproxyCache) Size() int64 {
	return gc.size
}

// ModTime implements the `goproxy.Cache`.
func (gc *goproxyCache) ModTime() time.Time {
	return gc.modTime
}

// Checksum implements the `goproxy.Cache`.
func (gc *goproxyCache) Checksum() []byte {
	return gc.checksum
}

// isAutoRedirectableGoproxyCache reports whether the name refers to an
// auto-redirectable Goproxy cache.
func isAutoRedirectableGoproxyCache(name string) bool {
	return !strings.HasPrefix(name, "sumdb/") &&
		strings.Contains(name, "/@v/") &&
		path.Ext(name) == ".zip"
}
