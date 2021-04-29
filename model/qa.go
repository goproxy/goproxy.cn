package model

import "html/template"

// QA is the question and answer model.
type QA struct {
	ID       string
	Question string
	Answer   template.HTML
}
