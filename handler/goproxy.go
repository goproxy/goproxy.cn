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
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/aofei/air"
	pgoproxy "github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/goproxy/goproxy/cacher"
)

var (
	// goproxyViper is used to get the configuration items of the Goproxy.
	goproxyViper = base.Viper.Sub("goproxy")

	// goproxy is an instance of the `pgoproxy.Goproxy`.
	goproxy = pgoproxy.New()

	// goproxyKodoCacher is the `cacher.Kodo` for the Qiniu Cloud Kodo.
	goproxyKodoCacher = &cacher.Kodo{
		Endpoint:   qiniuViper.GetString("kodo_endpoint"),
		AccessKey:  qiniuAccessKey,
		SecretKey:  qiniuSecretKey,
		BucketName: qiniuViper.GetString("kodo_bucket_name"),
	}

	// goproxyTimeout is the the maximum execution duration allowed for a Go
	// module proxy request.
	goproxyTimeout = goproxyViper.GetDuration("timeout")

	// goproxyAutoRedirect indicates whether the automatic redirection is
	// enabled for Go module proxy requests.
	goproxyAutoRedirect = goproxyViper.GetBool("auto_redirect")

	// qiniuKodoBucketEndpoint is the bucket endpoint for the Qiniu Cloud
	// Kodo.
	qiniuKodoBucketEndpoint = qiniuViper.GetString("kodo_bucket_endpoint")
)

func init() {
	ctx, cancel := context.WithCancel(context.Background())
	base.Air.AddShutdownJob(cancel)

	if err := goproxyViper.Unmarshal(goproxy); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to unmarshal goproxy configuration items")
	}

	if !goproxyViper.GetBool("disable_cacher") {
		goproxyLocalCacheRoot, err := ioutil.TempDir(
			goproxyViper.GetString("local_cache_root"),
			"",
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

		goproxy.Cacher = &goproxyCacher{
			Cacher:         goproxyKodoCacher,
			localCacheRoot: goproxyLocalCacheRoot,
			settingContext: ctx,
		}
	}

	goproxy.ErrorLogger = log.New(base.Logger, "", 0)

	base.Air.BATCH(nil, "/*", hGoproxy)
}

// hGoproxy handles requests to play with Go module proxy.
func hGoproxy(req *air.Request, res *air.Response) error {
	ctx, cancel := context.WithTimeout(req.Context, goproxyTimeout)
	defer cancel()

	req.Context = ctx

	name := strings.TrimLeft(path.Clean(req.RawPath()), "/")
	if !goproxyAutoRedirect || !isAutoRedirectableGoproxyCache(name) {
		goproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
		return nil
	}

	cache, err := goproxyKodoCacher.Cache(req.Context, name)
	if err != nil {
		if !errors.Is(err, pgoproxy.ErrCacheNotFound) {
			return err
		}

		goproxy.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

		return nil
	}

	cache.Close() // Just check for existence, no need to read

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
	pgoproxy.Cacher

	localCacheRoot    string
	settingContext    context.Context
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
				defer os.Remove(localCacheFile.Name())

				gc.settingCaches.Delete(k)

				cache := v.(pgoproxy.Cache)
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
func (gc *goproxyCacher) SetCache(ctx context.Context, c pgoproxy.Cache) error {
	gc.startSetCacheOnce.Do(gc.startSetCache)

	localCacheFile, err := ioutil.TempFile(gc.localCacheRoot, "")
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
