package handler

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/aofei/air"
	"github.com/fsnotify/fsnotify"
	"github.com/goproxy/goproxy.cn/base"
	"github.com/pelletier/go-toml/v2"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"golang.org/x/text/language"
)

var (
	// qas is the question and answers.
	qas []*qa

	// parseQAsOnce is used to guarantee that the `parseQAs` will only be
	// called once.
	parseQAsOnce = &sync.Once{}
)

// qa is the question and answer.
type qa struct {
	ID string

	languageMatcher language.Matcher
	questions       map[string]string
	answers         map[string]template.HTML
}

// Question returns the localized question for the locale.
func (qa *qa) Question(locale string) string {
	t, _ := language.MatchStrings(qa.languageMatcher, locale)
	return qa.questions[t.String()]
}

// Answer returns the localized answer for the locale.
func (qa *qa) Answer(locale string) template.HTML {
	t, _ := language.MatchStrings(qa.languageMatcher, locale)
	return qa.answers[t.String()]
}

func init() {
	qasWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to build qa watcher")
	} else if err := qasWatcher.Add("qas"); err != nil {
		base.Logger.Fatal().Err(err).
			Msg("failed to watch qa root")
	}

	go func() {
		done := make(chan struct{})
		base.Air.AddShutdownJob(func() {
			close(done)
		})

		for {
			select {
			case <-qasWatcher.Events:
				parseQAsOnce = &sync.Once{}
			case err := <-qasWatcher.Errors:
				base.Logger.Error().Err(err).
					Msg("qa watcher error")
			case <-done:
				return
			}
		}
	}()

	base.Air.BATCH(getHeadMethods, "/faq", hFAQPage)
}

// hFAQPage handles requests to get FAQ page.
func hFAQPage(req *air.Request, res *air.Response) error {
	parseQAsOnce.Do(parseQAs)
	return res.Render(map[string]any{
		"PageTitle":     req.LocalizedString("FAQ"),
		"CanonicalPath": "/faq",
		"IsFAQPage":     true,
		"QAs":           qas,
		"Locale":        req.Header.Get("Accept-Language"),
	}, "faq.html", "layouts/default.html")
}

// parseQAs parses frequently asked questions.
func parseQAs() {
	const qaRoot = "qas"

	qades, err := os.ReadDir(qaRoot)
	if err != nil {
		return
	}

	gm := goldmark.New(goldmark.WithExtensions(extension.GFM))

	nqas := map[string]*qa{}
	nqalts := map[string][]language.Tag{}
	for _, qade := range qades {
		if qade.IsDir() || filepath.Ext(qade.Name()) != ".md" {
			continue
		}

		n := qade.Name()
		n = strings.TrimSuffix(n, filepath.Ext(n))

		locale := strings.TrimPrefix(filepath.Ext(n), ".")
		if locale == "" {
			continue
		}

		tag, err := language.Parse(locale)
		if err != nil {
			continue
		}

		locale = tag.String()

		id := filepath.Base(strings.TrimSuffix(n, filepath.Ext(n)))
		if id == "" {
			continue
		}

		b, err := os.ReadFile(filepath.Join(qaRoot, qade.Name()))
		if err != nil {
			continue
		}

		if bytes.Count(b, []byte{'+', '+', '+'}) < 2 {
			continue
		}

		i := bytes.Index(b, []byte{'+', '+', '+'})
		j := bytes.Index(b[i+3:], []byte{'+', '+', '+'}) + 3

		md := map[string]string{}
		if err := toml.Unmarshal(b[i+3:j], &md); err != nil {
			continue
		}

		q := md["question"]
		if q == "" {
			continue
		}

		buf := bytes.Buffer{}
		if err := gm.Convert(b[j+3:], &buf); err != nil {
			continue
		}

		if _, ok := nqas[id]; !ok {
			nqas[id] = &qa{
				ID:        id,
				questions: map[string]string{},
				answers:   map[string]template.HTML{},
			}
		}

		nqas[id].questions[locale] = q
		nqas[id].answers[locale] = template.HTML(buf.String())
		nqalts[id] = append(nqalts[id], tag)
	}

	noqas := make([]*qa, 0, len(nqas))
	for _, nqa := range nqas {
		nqalts := nqalts[nqa.ID]
		sort.Slice(nqalts, func(i, j int) bool {
			return nqalts[i].String() == base.Air.I18nLocaleBase
		})
		nqa.languageMatcher = language.NewMatcher(nqalts)
		noqas = append(noqas, nqa)
	}

	sort.Slice(noqas, func(i, j int) bool {
		return noqas[i].ID < noqas[j].ID
	})

	qas = noqas
}
