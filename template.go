package tmpl

import (
	"context"
	"html/template"
	"io"
	"sync"
)

// TemplateProvider is a struct type that returns its corresponding template text.
type TemplateProvider = interface {
	TemplateText() string
}

type Template[T TemplateProvider] interface {
	Render(w io.Writer, data T, opts ...RenderOption) error
}

// tmpl represents a loaded and compiled tmpl file
type tmpl[T TemplateProvider] struct {
	// ctx is the template's compiler context
	ctx context.Context
	// mu is the mutex used to write to the underlying template
	mu *sync.RWMutex
	// p is the value of the TemplateProvider originally passed
	// to the Compile function
	p TemplateProvider
	// parser is the template's parser
	parser Parser
	// template is the compiled Go template
	template *template.Template
}
