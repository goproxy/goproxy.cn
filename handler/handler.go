package handler

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/goproxy/goproxy"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/goproxy/goproxy/cacher"
	"github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"
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

	// minioClient is the `minio.Client` for the Qiniu Cloud Kodo.
	minioClient *minio.Client

	// fileRemovals is a map of the files waiting to be removed.
	fileRemovals sync.Map

	// g is an instance of the `goproxy.Goproxy`.
	g = goproxy.New()
)

func init() {
	keu, err := url.Parse(cfg.Qiniu.KodoEndpoint)
	if err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to parse qiniu cloud kodo endpoint")
	}

	if minioClient, err = minio.NewWithOptions(
		strings.TrimPrefix(keu.String(), keu.Scheme+"://"),
		&minio.Options{
			Creds: credentials.NewStatic(
				cfg.Qiniu.AccessKey,
				cfg.Qiniu.SecretKey,
				"",
				credentials.SignatureDefault,
			),
			Secure:       strings.ToLower(keu.Scheme) == "https",
			BucketLookup: minio.BucketLookupPath,
		},
	); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to create minio client")
	}

	if _, err := cfg.Cron.AddFunc("*/10 * * * * *", func() {
		fileRemovals.Range(func(k, v interface{}) bool {
			if time.Now().Sub(v.(time.Time)) < 30*time.Second {
				return true
			}

			if err := minioClient.RemoveObject(
				cfg.Qiniu.KodoBucketName,
				k.(string),
			); err == nil {
				fileRemovals.Delete(k)
			}

			return true
		})
	}); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to add file removal cron job")
	}

	if eID, err := cfg.Cron.AddFunc("0 0 * * * *", func() {
		minioCore := &minio.Core{
			Client: minioClient,
		}

		var marker string
		for {
			lbr, err := minioCore.ListObjects(
				cfg.Qiniu.KodoBucketName,
				"",
				marker,
				"/@v/v",
				1000,
			)
			if err != nil {
				if err == io.EOF {
					break
				}

				panic(err)
			}

			for _, content := range lbr.Contents {
				if isFileMirrorable(content.Key) {
					continue
				}

				if err := minioClient.RemoveObject(
					cfg.Qiniu.KodoBucketName,
					content.Key,
				); err != nil {
					fileRemovals.Store(
						content.Key,
						time.Now().Add(30*time.Second),
					)
				}
			}

			if !lbr.IsTruncated {
				break
			}

			marker = lbr.NextMarker
		}
	}); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to add file corection cron job")
	} else {
		go cfg.Cron.Entry(eID).Job.Run() // Run once at the beginning
	}

	g.GoBinName = cfg.Goproxy.GoBinName
	g.MaxGoBinWorkers = cfg.Goproxy.MaxGoBinWorkers
	g.Cacher = &localCacher{
		Cacher: &cacher.Kodo{
			Endpoint:   cfg.Qiniu.KodoEndpoint,
			AccessKey:  cfg.Qiniu.AccessKey,
			SecretKey:  cfg.Qiniu.SecretKey,
			BucketName: cfg.Qiniu.KodoBucketName,
		},
		alwaysMissingCaches: cfg.Goproxy.AlwaysMissingCaches,
		localCacheRoot:      cfg.Goproxy.LocalCacheRoot,
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
	g.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())
	if res.Status >= http.StatusBadRequest {
		return nil
	}

	trimmedPath, _ := splitPathQuery(req.Path)
	trimmedPath = path.Clean(trimmedPath)
	trimmedPath = strings.TrimPrefix(trimmedPath, g.PathPrefix)
	trimmedPath = strings.TrimLeft(trimmedPath, "/")

	name, err := url.PathUnescape(trimmedPath)
	if err != nil {
		return err
	}

	if !isFileMirrorable(name) {
		fileRemovals.Store(name, time.Now())
	}

	return nil
}

// localCacher implements the `goproxy.Cacher`.
type localCacher struct {
	goproxy.Cacher
	alwaysMissingCaches bool
	localCacheRoot      string
}

func (lc *localCacher) Cache(
	ctx context.Context,
	name string,
) (goproxy.Cache, error) {
	if lc.alwaysMissingCaches {
		return nil, goproxy.ErrCacheNotFound
	}

	return lc.Cacher.Cache(ctx, name)
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

// isFileMirrorable reports whether the named file is mirrorable in the Qiniu
// Cloud Kodo.
func isFileMirrorable(name string) bool {
	switch {
	case strings.HasPrefix(name, "sumdb/"):
		if strings.HasSuffix(name, "/supported") ||
			strings.HasSuffix(name, "/latest") {
			return false
		}
	case strings.HasSuffix(name, "/@latest"),
		strings.HasSuffix(name, "/@v/list"):
		return false
	case strings.Contains(name, "/@"):
		emv := strings.TrimSuffix(path.Base(name), path.Ext(name))
		mv, err := module.UnescapeVersion(emv)
		if err != nil {
			return false
		}

		if !semver.IsValid(mv) {
			return false
		}
	}

	return true
}
