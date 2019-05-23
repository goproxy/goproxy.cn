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
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/air-gases/cacheman"
	"github.com/aofei/air"
	"github.com/cespare/xxhash/v2"
	"github.com/goproxy/goproxy.cn/cfg"
	"github.com/qiniu/api.v7/auth/qbox"
	"github.com/qiniu/api.v7/storage"
	"github.com/rs/zerolog/log"
	"golang.org/x/mod/module"
)

var (
	// a is the `air.Default`.
	a = air.Default

	qiniuMac                  *qbox.Mac
	qiniuStorageConfig        *storage.Config
	qiniuStorageBucketManager *storage.BucketManager

	goBinWorkerChan = make(chan struct{}, cfg.Goproxy.MaxGoBinWorkers)

	modOutputNotFoundKeywords = [][]byte{
		[]byte("could not read username"),
		[]byte("invalid"),
		[]byte("malformed"),
		[]byte("no matching"),
		[]byte("not found"),
		[]byte("unknown"),
		[]byte("unrecognized"),
	}

	errModuleNotFound = errors.New("module not found")
)

func init() {
	qiniuMac = qbox.NewMac(cfg.Qiniu.AccessKey, cfg.Qiniu.SecretKey)

	qiniuStorageRegion, err := storage.GetRegion(
		cfg.Qiniu.AccessKey,
		cfg.Qiniu.StorageBucket,
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

	if err := os.Setenv("GO111MODULE", "on"); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to set $GO111MODULE")
	}

	if err := os.Setenv("GOPROXY", "direct"); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to set $GOPROXY")
	}

	if err := os.Setenv("GOSUMDB", "off"); err != nil {
		log.Fatal().Err(err).
			Str("app_name", a.AppName).
			Msg("failed to set $GOSUMDB")
	}

	a.BATCH(
		[]string{http.MethodGet, http.MethodHead},
		"/",
		indexPageHandler,
	)
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

// indexPageHandler handles requests to get index page.
func indexPageHandler(req *air.Request, res *air.Response) error {
	return res.Redirect("https://github.com/goproxy/goproxy.cn")
}

// goproxyHandler handles requests to perform a Go module proxy action.
func goproxyHandler(req *air.Request, res *air.Response) error {
	filename := req.Param("*").Value().String()
	filenameParts := strings.Split(filename, "/@")
	if len(filenameParts) != 2 {
		return a.NotFoundHandler(req, res)
	}

	switch filenameParts[1] {
	case "latest":
		mlr, err := modList(req.Context, filenameParts[0], false)
		if err != nil {
			if err == errModuleNotFound {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		return res.WriteJSON(mlr)
	case "v/list":
		mlr, err := modList(req.Context, filenameParts[0], true)
		if err != nil {
			if err == errModuleNotFound {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		return res.WriteString(strings.Join(mlr.Versions, "\n"))
	}

	fileInfo, err := qiniuStorageBucketManager.Stat(
		cfg.Qiniu.StorageBucket,
		filename,
	)
	if err != nil && err.Error() == "no such file or directory" {
		filenameBase := path.Base(filenameParts[1])
		filenameExt := path.Ext(filenameBase)
		switch filenameExt {
		case ".info", ".mod", ".zip":
		default:
			return a.NotFoundHandler(req, res)
		}

		if _, err := modDownload(
			req.Context,
			filenameParts[0],
			strings.TrimSuffix(filenameBase, filenameExt),
		); err != nil {
			if err == errModuleNotFound {
				return a.NotFoundHandler(req, res)
			}

			return err
		}

		if fileInfo, err = qiniuStorageBucketManager.Stat(
			cfg.Qiniu.StorageBucket,
			filename,
		); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	fileReq, err := http.NewRequest(
		http.MethodGet,
		storage.MakePrivateURL(
			qiniuMac,
			cfg.Qiniu.StorageBucketAccessEndpoint,
			filename,
			time.Now().Add(time.Hour).Unix(),
		),
		nil,
	)
	if err != nil {
		return err
	}

	fileReq = fileReq.WithContext(req.Context)

	fileRes, err := http.DefaultClient.Do(fileReq)
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
	Version  string   `json:"Version"`
	Time     string   `json:"Time"`
	Versions []string `json:"Versions,omitempty"`
}

// modList executes `go list -json -m -versions modulePath@latest`.
func modList(
	ctx context.Context,
	escapedModulePath string,
	allVersions bool,
) (*modListResult, error) {
	modulePath, err := module.UnescapePath(escapedModulePath)
	if err != nil {
		return nil, errModuleNotFound
	}

	goproxyRoot, err := ioutil.TempDir("", "goproxy")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(goproxyRoot)

	args := []string{"list", "-json", "-m"}
	if allVersions {
		args = append(args, "-versions")
	}

	args = append(args, fmt.Sprint(modulePath, "@latest"))

	stdout, err := executeGoCommand(ctx, goproxyRoot, args...)
	if err != nil {
		return nil, err
	}

	mlr := &modListResult{}
	if err := json.Unmarshal(stdout, mlr); err != nil {
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
func modDownload(
	ctx context.Context,
	escapedModulePath string,
	escapedModuleVersion string,
) (*modDownloadResult, error) {
	modulePath, err := module.UnescapePath(escapedModulePath)
	if err != nil {
		return nil, errModuleNotFound
	}

	moduleVersion, err := module.UnescapeVersion(escapedModuleVersion)
	if err != nil {
		return nil, errModuleNotFound
	}

	goproxyRoot, err := ioutil.TempDir("", "goproxy")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(goproxyRoot)

	stdout, err := executeGoCommand(
		ctx,
		goproxyRoot,
		"mod",
		"download",
		"-json",
		fmt.Sprint(modulePath, "@", moduleVersion),
	)
	if err != nil {
		return nil, err
	}

	mdr := &modDownloadResult{}
	if err := json.Unmarshal(stdout, mdr); err != nil {
		return nil, err
	}

	filenamePrefix := path.Join(
		escapedModulePath,
		"@v",
		escapedModuleVersion,
	)

	infoFilename := fmt.Sprint(filenamePrefix, ".info")
	if err := uploadFile(
		ctx,
		infoFilename,
		mdr.Info,
		"application/json; charset=utf-8",
	); err != nil {
		return nil, err
	}

	modFilename := fmt.Sprint(filenamePrefix, ".mod")
	if err := uploadFile(
		ctx,
		modFilename,
		mdr.GoMod,
		"text/plain; charset=utf-8",
	); err != nil {
		return nil, err
	}

	zipFilename := fmt.Sprint(filenamePrefix, ".zip")
	if err := uploadFile(
		ctx,
		zipFilename,
		mdr.Zip,
		"application/zip",
	); err != nil {
		return nil, err
	}

	return mdr, nil
}

// executeGoCommand executes go command with the args.
func executeGoCommand(
	ctx context.Context,
	goproxyRoot string,
	args ...string,
) ([]byte, error) {
	goBinWorkerChan <- struct{}{}
	defer func() {
		<-goBinWorkerChan
	}()

	cmd := exec.CommandContext(ctx, cfg.Goproxy.GoBinName, args...)
	cmd.Env = append(
		os.Environ(),
		fmt.Sprint("GOCACHE=", filepath.Join(goproxyRoot, "gocache")),
		fmt.Sprint("GOPATH=", filepath.Join(goproxyRoot, "gopath")),
	)
	cmd.Dir = goproxyRoot
	stdout, err := cmd.Output()
	if err != nil {
		output := stdout
		if ee, ok := err.(*exec.ExitError); ok {
			output = append(output, ee.Stderr...)
		}

		lowercasedOutput := bytes.ToLower(output)
		for _, k := range modOutputNotFoundKeywords {
			if bytes.Contains(lowercasedOutput, k) {
				return nil, errModuleNotFound
			}
		}

		return nil, fmt.Errorf("modList: %v: %s", err, output)
	}

	return stdout, nil
}

// uploadFile uploads the localFilename as the contentType to the Qiniu storage
// bucket. The filename is the new name in the Qiniu storage bucket.
func uploadFile(
	ctx context.Context,
	filename string,
	localFilename string,
	contentType string,
) error {
	return storage.NewFormUploader(qiniuStorageConfig).PutFile(
		ctx,
		nil,
		(&storage.PutPolicy{
			Scope: fmt.Sprintf(
				"%s:%s",
				cfg.Qiniu.StorageBucket,
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
