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

	// kodoConfig is the configuration of the Qiniu Cloud Kodo.
	kodoConfig *storage.Config

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
	g.Cacher = &cacher{}
	g.ErrorLogger = a.ErrorLogger

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

	kodoConfig = &storage.Config{
		Region: kodoRegion,
	}

	kodoBucketManager = storage.NewBucketManager(kodoMac, kodoConfig)

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
				cfg.Kodo.Endpoint,
				fk,
				time.Now().Add(time.Hour).Unix(),
			)
			return res.Redirect(fu)
		}
	}

	g.ServeHTTP(res.HTTPResponseWriter(), req.HTTPRequest())

	return nil
}

// cacher implements the `goproxy.Cacher`.
type cacher struct{}

// NewHash implements the `goproxy.Cacher`.
func (*cacher) NewHash() hash.Hash {
	return md5.New()
}

// Cache implements the `goproxy.Cacher`.
func (*cacher) Cache(ctx context.Context, name string) (goproxy.Cache, error) {
	fileInfo, err := kodoBucketManager.Stat(cfg.Kodo.BucketName, name)
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

	fileURL := storage.MakePrivateURL(
		kodoMac,
		cfg.Kodo.Endpoint,
		name,
		deadline.Unix(),
	)

	req, err := http.NewRequest(http.MethodHead, fileURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	checksum, err := hex.DecodeString(res.Header.Get("x-qn-meta-checksum"))
	if err != nil {
		return nil, err
	}

	return &cache{
		ctx:      ctx,
		url:      fileURL,
		name:     name,
		mimeType: fileInfo.MimeType,
		size:     fileInfo.Fsize,
		modTime:  time.Unix(fileInfo.PutTime*100/int64(time.Second), 0),
		checksum: checksum,
	}, nil
}

// SetCache implements the `goproxy.Cacher`.
func (*cacher) SetCache(ctx context.Context, c goproxy.Cache) error {
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

		scope := fmt.Sprintf("%s:%s", cfg.Kodo.BucketName, c.Name())
		checksumString := hex.EncodeToString(c.Checksum())
		storage.NewFormUploader(kodoConfig).PutFile(
			ctx,
			nil,
			(&storage.PutPolicy{
				Scope: scope,
			}).UploadToken(kodoMac),
			c.Name(),
			localCache.Name(),
			&storage.PutExtra{
				Params: map[string]string{
					"x-qn-meta-checksum": checksumString,
				},
				MimeType: c.MIMEType(),
			},
		)
	}()

	return nil
}

// cache implements the `goproxy.Cache`. It is the cache unit of the `cacher`.
type cache struct {
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
func (c *cache) Read(b []byte) (int, error) {
	if c.closed {
		return 0, os.ErrClosed
	} else if c.offset >= c.size {
		return 0, io.EOF
	}

	req, err := http.NewRequest(http.MethodGet, c.url, nil)
	if err != nil {
		return 0, err
	}

	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", c.offset))

	res, err := http.DefaultClient.Do(req.WithContext(c.ctx))
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	n, err := res.Body.Read(b)
	c.offset += int64(n)

	return n, err
}

// Seek implements the `goproxy.Cache`.
func (c *cache) Seek(offset int64, whence int) (int64, error) {
	if c.closed {
		return 0, os.ErrClosed
	}

	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += c.offset
	case io.SeekEnd:
		offset += c.size
	default:
		return 0, errors.New("invalid whence")
	}

	if offset < 0 {
		return 0, errors.New("negative position")
	}

	c.offset = offset

	return c.offset, nil
}

// Close implements the `goproxy.Cache`.
func (c *cache) Close() error {
	if c.closed {
		return os.ErrClosed
	}

	c.closed = true

	return nil
}

// Name implements the `goproxy.Cache`.
func (c *cache) Name() string {
	return c.name
}

// MIMEType implements the `goproxy.Cache`.
func (c *cache) MIMEType() string {
	return c.mimeType
}

// Size implements the `goproxy.Cache`.
func (c *cache) Size() int64 {
	return c.size
}

// ModTime implements the `goproxy.Cache`.
func (c *cache) ModTime() time.Time {
	return c.modTime
}

// Checksum implements the `goproxy.Cache`.
func (c *cache) Checksum() []byte {
	return c.checksum
}

// isKodoFileNotExist reports whether the err means a Kodo file is not exist.
func isKodoFileNotExist(err error) bool {
	return err != nil && err.Error() == "no such file or directory"
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
