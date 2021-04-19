package handler

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"hash"
	"io"
	"io/fs"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
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
	hhGoproxy = goproxy.New()

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

	goproxyLocalCacheRoot, err := os.MkdirTemp(
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
		localCacheRoot: goproxyLocalCacheRoot,
	}

	hhGoproxy.ErrorLogger = log.New(base.Logger, "", 0)

	base.Air.BATCH(getHeadMethods, "/*", hGoproxy)
}

// hGoproxy handles requests to play with Go module proxy.
func hGoproxy(req *air.Request, res *air.Response) error {
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
type goproxyCacher struct {
	localCacheRoot    string
	settingMutex      sync.Mutex
	settingCaches     sync.Map
	startSetCacheOnce sync.Once
}

// startSetCache starts the cache setting of the gc.
func (gc *goproxyCacher) startSetCache() {
	go func() {
		for {
			time.Sleep(time.Second)
			if base.Context.Err() != nil {
				return
			}

			gc.settingCaches.Range(func(k, v interface{}) bool {
				if base.Context.Err() != nil {
					return false
				}

				localCacheFile, err := os.Open(k.(string))
				if err != nil {
					if errors.Is(err, fs.ErrNotExist) {
						gc.settingCaches.Delete(k)
					}

					return true
				}
				defer localCacheFile.Close()

				cache := v.(goproxy.Cache)
				if _, err := qiniuKodoClient.StatObject(
					base.Context,
					qiniuKodoBucketName,
					cache.Name(),
					minio.StatObjectOptions{},
				); err == nil {
					gc.settingCaches.Delete(k)
					gc.settingMutex.Lock()
					os.Remove(localCacheFile.Name())
					gc.settingMutex.Unlock()
					return true
				} else if !isMinIOObjectNotExist(err) {
					return true
				}

				if _, err := qiniuKodoClient.PutObject(
					base.Context,
					qiniuKodoBucketName,
					cache.Name(),
					localCacheFile,
					cache.Size(),
					minio.PutObjectOptions{
						ContentType:      cache.MIMEType(),
						DisableMultipart: cache.Size() < 256<<20,
					},
				); err == nil {
					gc.settingCaches.Delete(k)
					gc.settingMutex.Lock()
					os.Remove(localCacheFile.Name())
					gc.settingMutex.Unlock()
				}

				return true
			})
		}
	}()
}

// NewHash implements the `goproxy.Cacher`.
func (gc *goproxyCacher) NewHash() hash.Hash {
	return md5.New()
}

// Cache implements the `goproxy.Cacher`.
func (gc *goproxyCacher) Cache(
	ctx context.Context,
	name string,
) (goproxy.Cache, error) {
	objectInfo, err := qiniuKodoClient.StatObject(
		ctx,
		qiniuKodoBucketName,
		name,
		minio.StatObjectOptions{},
	)
	if err != nil {
		if isMinIOObjectNotExist(err) {
			return nil, goproxy.ErrCacheNotFound
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

	return &goproxyCache{
		ReadSeekCloser: object,
		name:           name,
		mimeType:       objectInfo.ContentType,
		size:           objectInfo.Size,
		modTime:        objectInfo.LastModified,
		checksum:       checksum,
	}, nil
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
	} else if !errors.Is(err, fs.ErrNotExist) {
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
	io.ReadSeekCloser

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
