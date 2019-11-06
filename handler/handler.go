package handler

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
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
	g.Cacher = &kodoCacher{
		bucketName:     cfg.Qiniu.KodoBucketName,
		bucketEndpoint: cfg.Qiniu.KodoBucketEndpoint,
		bucketManager:  qiniuKodoBucketManager,
		formUploader:   storage.NewFormUploader(qiniuKodoConfig),
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

// kodoCacher implements the `goproxy.Cacher`.
type kodoCacher struct {
	bucketName     string
	bucketEndpoint string
	bucketManager  *storage.BucketManager
	formUploader   *storage.FormUploader
	localCacheRoot string
}

// NewHash implements the `goproxy.Cacher`.
func (kc *kodoCacher) NewHash() hash.Hash {
	return md5.New()
}

// Cache implements the `goproxy.Cacher`.
func (kc *kodoCacher) Cache(
	ctx context.Context,
	name string,
) (goproxy.Cache, error) {
	fi, err := kc.bucketManager.Stat(kc.bucketName, name)
	if err != nil {
		if isKodoFileNotExist(err) {
			return nil, goproxy.ErrCacheNotFound
		}

		return nil, err
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		deadline = time.Now().Add(time.Hour)
	}

	url := storage.MakePrivateURL(
		kc.bucketManager.Mac,
		kc.bucketEndpoint,
		name,
		deadline.Unix(),
	)

	req, err := http.NewRequest(http.MethodHead, url, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	checksum, err := hex.DecodeString(res.Header.Get("X-QN-Meta-Checksum"))
	if err != nil {
		return nil, err
	}

	return &kodoCache{
		ctx: ctx,
		url: storage.MakePrivateURL(
			kc.bucketManager.Mac,
			kc.bucketEndpoint,
			name,
			deadline.Unix(),
		),
		name:     name,
		mimeType: fi.MimeType,
		size:     fi.Fsize,
		modTime:  storage.ParsePutTime(fi.PutTime),
		checksum: checksum,
	}, nil
}

// SetCache implements the `goproxy.Cacher`.
func (kc *kodoCacher) SetCache(ctx context.Context, c goproxy.Cache) error {
	localCache, err := ioutil.TempFile(kc.localCacheRoot, "")
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

		kc.formUploader.PutFile(
			ctx,
			&storage.PutRet{},
			(&storage.PutPolicy{
				Scope: kc.bucketName,
			}).UploadToken(kc.bucketManager.Mac),
			c.Name(),
			localCache.Name(),
			&storage.PutExtra{
				Params: map[string]string{
					"x-qn-meta-checksum": hex.
						EncodeToString(c.Checksum()),
				},
				MimeType: c.MIMEType(),
			},
		)
	}()

	return nil
}

// kodoCache implements the `goproxy.Cache`. It is the cache unit of the
// `kodoCacher`.
type kodoCache struct {
	ctx      context.Context
	url      string
	offset   int64
	closed   bool
	name     string
	mimeType string
	size     int64
	modTime  time.Time
	checksum []byte
}

// Read implements the `goproxy.Cache`.
func (kc *kodoCache) Read(b []byte) (int, error) {
	if kc.closed {
		return 0, os.ErrClosed
	} else if kc.offset >= kc.size {
		return 0, io.EOF
	}

	req, err := http.NewRequest(http.MethodGet, kc.url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", kc.offset))

	res, err := http.DefaultClient.Do(req.WithContext(kc.ctx))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	n, err := res.Body.Read(b)
	kc.offset += int64(n)

	return n, err
}

// Seek implements the `goproxy.Cache`.
func (kc *kodoCache) Seek(offset int64, whence int) (int64, error) {
	if kc.closed {
		return 0, os.ErrClosed
	}

	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += kc.offset
	case io.SeekEnd:
		offset += kc.size
	default:
		return 0, errors.New("invalid whence")
	}

	if offset < 0 {
		return 0, errors.New("negative position")
	}

	kc.offset = offset

	return kc.offset, nil
}

// Close implements the `goproxy.Cache`.
func (kc *kodoCache) Close() error {
	if kc.closed {
		return os.ErrClosed
	}

	kc.closed = true

	return nil
}

// Name implements the `goproxy.Cache`.
func (kc *kodoCache) Name() string {
	return kc.name
}

// MIMEType implements the `goproxy.Cache`.
func (kc *kodoCache) MIMEType() string {
	return kc.mimeType
}

// Size implements the `goproxy.Cache`.
func (kc *kodoCache) Size() int64 {
	return kc.size
}

// ModTime implements the `goproxy.Cache`.
func (kc *kodoCache) ModTime() time.Time {
	return kc.modTime
}

// Checksum implements the `goproxy.Cache`.
func (kc *kodoCache) Checksum() []byte {
	return kc.checksum
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
