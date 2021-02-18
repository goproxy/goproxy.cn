package handler

import (
	"bytes"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/aofei/air"
	"github.com/fsnotify/fsnotify"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/goproxy/goproxy.cn/model"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

var (
	// faqs is the frequently asked questions.
	faqs = map[string][]model.QA{}

	// parseFAQsOnce is used to guarantee that the `parseFAQs` will only be
	// called once.
	parseFAQsOnce = &sync.Once{}
)

func init() {
	faqsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to build faq watcher")
	} else if err := faqsWatcher.Add("faqs"); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to watch faq directory")
	}

	if err := filepath.WalkDir(
		"faqs",
		func(p string, de fs.DirEntry, err error) error {
			if de == nil || !de.IsDir() {
				return err
			}

			return faqsWatcher.Add(p)
		},
	); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to watch faq directory")
	}

	go func() {
		done := make(chan struct{})
		base.Air.AddShutdownJob(func() {
			close(done)
		})

		for {
			select {
			case <-faqsWatcher.Events:
				parseFAQsOnce = &sync.Once{}
			case err := <-faqsWatcher.Errors:
				base.Logger.Error().Err(err).
					Msg("faq watcher error")
			case <-done:
				return
			}
		}
	}()

	base.Air.BATCH(getHeadMethods, "/faq", hFaqPage)
}

// hFaqPage handles requests to get FAQ page.
func hFaqPage(req *air.Request, res *air.Response) error {
	parseFAQsOnce.Do(parseFAQs)

	qas, ok := faqs[req.LocalizedString("FAQ")]
	if !ok {
		qas = faqs["FAQ"]
	}

	return res.Render(map[string]interface{}{
		"PageTitle":     req.LocalizedString("FAQ"),
		"CanonicalPath": "/faq",
		"IsFAQPage":     true,
		"QAs":           qas,
	}, "faq.html", "layouts/default.html")
}

// parseFAQs parses frequently asked questions.
func parseFAQs() {
	var dirs []string
	if err := filepath.WalkDir(
		"faqs",
		func(p string, de fs.DirEntry, err error) error {
			if de == nil || !de.IsDir() {
				return err
			}

			dirs = append(dirs, p)

			return nil
		},
	); err != nil {
		return
	}

	gm := goldmark.New(goldmark.WithExtensions(extension.GFM))

	nfaqs := make(map[string][]model.QA, len(dirs))
	for _, dir := range dirs {
		var qas []model.QA
		if err := filepath.WalkDir(
			dir,
			func(p string, de fs.DirEntry, err error) error {
				if de == nil || de.IsDir() {
					return err
				}

				ext := filepath.Ext(p)
				if ext != ".md" {
					return err
				}

				b, err := os.ReadFile(p)
				if err != nil {
					return err
				}

				var buf bytes.Buffer
				if err := gm.Convert(b, &buf); err != nil {
					return err
				}

				qas = append(qas, model.QA{
					Question: strings.TrimSuffix(
						filepath.Base(p),
						ext,
					),
					Answer: template.HTML(buf.String()),
				})

				return nil
			},
		); err != nil {
			return
		}

		sort.Slice(qas, func(i, j int) bool {
			return qas[i].Question < qas[j].Question
		})

		for i := range qas {
			q := qas[i].Question
			q = q[strings.Index(q, "-")+1:]
			qas[i].Question = q
		}

		nfaqs[filepath.Base(dir)] = qas
	}

	faqs = nfaqs
}
