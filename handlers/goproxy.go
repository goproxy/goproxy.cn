package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/aofei/goproxy.cn/cfg"
	"github.com/cespare/xxhash/v2"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"github.com/rs/zerolog/log"
)

var (
	localCacheMutex     sync.Mutex
	localCacheWaitGroup sync.WaitGroup

	qiniuMac                  *qbox.Mac
	qiniuStorageConfig        *storage.Config
	qiniuStorageBucketManager *storage.BucketManager

	errModuleNotFound = errors.New("module not found")
)

func init() {
	qiniuMac = qbox.NewMac(
		cfg.Goproxy.QiniuAccessKey,
		cfg.Goproxy.QiniuSecretKey,
	)

	qiniuStorageRegion, err := storage.GetRegion(
		cfg.Goproxy.QiniuAccessKey,
		cfg.Goproxy.QiniuStorageBucket,
	)
	if err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to get qiniu storage region client")
	}

	qiniuStorageConfig = &storage.Config{
		Region: qiniuStorageRegion,
	}

	qiniuStorageBucketManager = storage.NewBucketManager(
		qiniuMac,
		qiniuStorageConfig,
	)

	goproxyRoot := filepath.Join(os.TempDir(), "goproxy")

	if err := os.Setenv(
		"GOCACHE",
		filepath.Join(goproxyRoot, "gocache"),
	); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to set $GOCACHE")
	}

	if err := os.Setenv(
		"GOPATH",
		filepath.Join(goproxyRoot, "gopath"),
	); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to set $GOPATH")
	}

	go func() {
		for {
			startTime := time.Now()

			var totalSize int64
			filepath.Walk(goproxyRoot, func(
				_ string,
				fi os.FileInfo,
				err error,
			) error {
				if fi != nil && !fi.IsDir() {
					totalSize += fi.Size()
				}

				return err
			})

			if totalSize > int64(cfg.Goproxy.MaxLocalCacheBytes) {
				localCacheMutex.Lock()

				localCacheWaitGroup.Wait()
				os.RemoveAll(goproxyRoot)

				localCacheMutex.Unlock()
			}

			if d := time.Now().Sub(startTime); d < 10*time.Minute {
				time.Sleep(10*time.Minute - d)
			}
		}
	}()

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/*",
		goproxyHandler,
		cacheman.Gas(cacheman.GasConfig{
			MustRevalidate: true,
			NoCache:        true,
			NoStore:        true,
			MaxAge:         -1,
			SMaxAge:        -1,
		}),
	)
}

// goproxyHandler handles requests to perform a Go module proxy action.
func goproxyHandler(req *air.Request, res *air.Response) error {
	localCacheMutex.Lock()
	localCacheMutex.Unlock()

	localCacheWaitGroup.Add(1)
	defer localCacheWaitGroup.Done()

	var (
		encodedFilename = req.Param("*").Value().String()
		filenameBuilder strings.Builder
		bang            bool
	)

	filenameBuilder.Grow(len(encodedFilename))
	for _, r := range encodedFilename {
		if r >= 'A' && r <= 'Z' {
			return a.NotFoundHandler(req, res)
		}

		if r == '!' {
			bang = true
			continue
		}

		if bang {
			bang = false
			if r >= 'a' && r <= 'z' {
				r -= 'a' - 'A' // To upper
			} else {
				filenameBuilder.WriteByte('!')
			}
		}

		filenameBuilder.WriteRune(r)
	}

	filename := filenameBuilder.String()
	filenameParts := strings.Split(filename, "/@")
	if len(filenameParts) != 2 {
		return a.NotFoundHandler(req, res)
	}

	modulePath := filenameParts[0]

	switch filenameParts[1] {
	case "v/list", "latest":
		mlo, err := modList(modulePath)
		if err != nil {
			if err == errModuleNotFound {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		switch filenameParts[1] {
		case "v/list":
			return res.WriteString(strings.Join(mlo.Versions, "\n"))
		case "latest":
			mlo.Versions = nil // No need
			return res.WriteJSON(mlo)
		}
	}

	fileInfo, err := qiniuStorageBucketManager.Stat(
		cfg.Goproxy.QiniuStorageBucket,
		filename,
	)
	if isFileNotExist(err) {
		filenameBase := path.Base(filenameParts[1])
		filenameExt := path.Ext(filenameBase)
		moduleVersion := strings.TrimSuffix(filenameBase, filenameExt)

		mdr, err := modDownload(modulePath, moduleVersion)
		if err != nil {
			if err == errModuleNotFound {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		director := path.Join(modulePath, "@v")

		infoFilename := path.Join(director, path.Base(mdr.Info))
		infoFileInfo, err := qiniuStorageBucketManager.Stat(
			cfg.Goproxy.QiniuStorageBucket,
			infoFilename,
		)
		if isFileNotExist(err) {
			if err := uploadFile(
				infoFilename,
				mdr.Info,
				"application/json; charset=utf-8",
			); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		modFilename := path.Join(director, path.Base(mdr.GoMod))
		modFileInfo, err := qiniuStorageBucketManager.Stat(
			cfg.Goproxy.QiniuStorageBucket,
			modFilename,
		)
		if isFileNotExist(err) {
			if err := uploadFile(
				modFilename,
				mdr.GoMod,
				"text/plain; charset=utf-8",
			); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		zipFilename := path.Join(director, path.Base(mdr.Zip))
		zipFileInfo, err := qiniuStorageBucketManager.Stat(
			cfg.Goproxy.QiniuStorageBucket,
			zipFilename,
		)
		if isFileNotExist(err) {
			if err := uploadFile(
				zipFilename,
				mdr.Zip,
				"application/zip",
			); err != nil {
				return err
			}
		} else if err != nil {
			return err
		}

		switch filenameExt {
		case path.Ext(mdr.Info):
			filename = infoFilename
			fileInfo = infoFileInfo
		case path.Ext(mdr.GoMod):
			filename = modFilename
			fileInfo = modFileInfo
		case path.Ext(mdr.Zip):
			filename = zipFilename
			fileInfo = zipFileInfo
		default:
			return a.NotFoundHandler(req, res)
		}

		if fileInfo.Hash == "" {
			if fileInfo, err = qiniuStorageBucketManager.Stat(
				cfg.Goproxy.QiniuStorageBucket,
				filename,
			); err != nil {
				return err
			}
		}
	} else if err != nil {
		return err
	}

	fileRes, err := http.Get(storage.MakePrivateURL(
		qiniuMac,
		cfg.Goproxy.QiniuStorageBucketAccessEndpoint,
		filename,
		time.Now().Add(time.Hour).Unix(),
	))
	if err != nil {
		return err
	}
	defer fileRes.Body.Close()

	res.Header.Set("Content-Type", fileInfo.MimeType)
	res.Header.Set("Content-Length", strconv.FormatInt(fileInfo.Fsize, 10))

	eTag := make([]byte, 8)
	binary.BigEndian.PutUint64(eTag, xxhash.Sum64String(fileInfo.Hash))
	res.Header.Set("ETag", fmt.Sprintf(
		"%q",
		base64.StdEncoding.EncodeToString(eTag),
	))

	res.Header.Set(
		"Last-Modified",
		storage.ParsePutTime(fileInfo.PutTime).Format(http.TimeFormat),
	)

	if path.Base(filename) == path.Base(req.Path) {
		res.Header.Set("Cache-Control", "max-age=31536000")
	}

	_, err = io.Copy(res.Body, fileRes.Body)

	return err
}

// modListResult is the result of
// `go list -json -m -versions <MODULE_PATH>@latest`.
type modListResult struct {
	Versions []string `json:"Versions,omitempty"`
	Version  string   `json:"Version"`
	Time     string   `json:"Time"`
}

// modList executes `go list -json -m -versions modulePath@latest`.
func modList(modulePath string) (*modListResult, error) {
	cmd := exec.Command(
		cfg.Goproxy.GoBinName,
		"list",
		"-json",
		"-m",
		"-versions",
		modulePath+"@latest",
	)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprint(stdout.String(), stderr.String())
		if strings.Contains(errMsg, "malformed module path") ||
			strings.Contains(errMsg, "invalid version") ||
			strings.Contains(errMsg, "unknown revision") {
			return nil, errModuleNotFound
		}

		return nil, err
	}

	mlr := &modListResult{}
	if err := json.Unmarshal(stdout.Bytes(), mlr); err != nil {
		return nil, err
	}

	return mlr, nil
}

// modDownloadResult is the result of
// `go mod download -json <MODULE_PATH>@<MODULE_VERSION>`.
type modDownloadResult struct {
	Info  string `json:"Info"`
	GoMod string `json:"GoMod"`
	Zip   string `json:"Zip"`
}

// modDownload executes `go mod download -json modulePath@moduleVersion`.
func modDownload(modulePath, moduleVersion string) (*modDownloadResult, error) {
	cmd := exec.Command(
		cfg.Goproxy.GoBinName,
		"mod",
		"download",
		"-json",
		modulePath+"@"+moduleVersion,
	)
	stdout := bytes.Buffer{}
	stderr := bytes.Buffer{}
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		errMsg := fmt.Sprint(stdout.String(), stderr.String())
		if strings.Contains(errMsg, "malformed module path") ||
			strings.Contains(errMsg, "invalid version") ||
			strings.Contains(errMsg, "unknown revision") {
			return nil, errModuleNotFound
		}

		return nil, err
	}

	mdr := &modDownloadResult{}
	if err := json.Unmarshal(stdout.Bytes(), mdr); err != nil {
		return nil, err
	}

	return mdr, nil
}

// isFileNotExist reports whether the err indicates that some file does not
// exist.
func isFileNotExist(err error) bool {
	return err != nil && err.Error() == "no such file or directory"
}

// uploadFile uploads the localFilename as the contentType to the Qiniu storage
// bucket. The filename is the new name in the Qiniu storage bucket.
func uploadFile(filename, localFilename, contentType string) error {
	return storage.NewFormUploader(qiniuStorageConfig).PutFile(
		context.Background(),
		nil,
		(&storage.PutPolicy{
			Scope: fmt.Sprintf(
				"%s:%s",
				cfg.Goproxy.QiniuStorageBucket,
				filename,
			),
		}).UploadToken(qiniuMac),
		filename,
		localFilename,
		&storage.PutExtra{
			MimeType: contentType,
		},
	)
}
