package tmpl

import (
	"html/template"
	"io"
	"sync"
)

// TemplateProvider is a struct type that returns its corresponding template text.
type TemplateProvider interface {
	TemplateText() string
}

type Template[T TemplateProvider] interface {
	// Render can be used to execute the internal template.
	Render(w io.Writer, data T, opts ...RenderOption) error
	// RenderToChan can be used to execute the internal template and write the result to a channel.
	RenderToChan(ch chan string, data T, opts ...RenderOption) error
	// RenderToString can be used to execute the internal template and return the result as a string.
	RenderToString(data T, opts ...RenderOption) (string, error)
}

// managedTemplate represents a loaded and compiled tmpl file
type managedTemplate[T TemplateProvider] struct {
	// mu is the mutex used to write to the underlying template
	mu *sync.RWMutex
	// template is the compiled Go template
	template *template.Template
}
