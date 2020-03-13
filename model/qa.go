package model

import "html/template"

// QA is the question and answer model.
type QA struct {
	Question string
	Answer   template.HTML
}
