package tmpl

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"strings"
	"sync"
)

// TemplateProvider is a struct type that returns its corresponding template text.
type TemplateProvider interface {
	TemplateText() string
}

func nameFromProvider(p TemplateProvider) string {
	return strings.TrimPrefix(fmt.Sprintf("%T", p), "*")
}

type Template[T TemplateProvider] interface {
	// Render can be used to execute the internal template.
	Render(w io.Writer, data T, opts ...RenderOption) error
}

// tmpl represents a loaded and compiled tmpl file
type tmpl[T TemplateProvider] struct {
	// ctx is the template's compiler context
	ctx context.Context
	// mu is the mutex used to write to the underlying template
	mu *sync.RWMutex
	// name is the name of the root template definition
	name string
	// template is the compiled Go template
	template *template.Template
	// signal represents the channel that will be notified when
	// the internal template has been recompiled
	signal chan struct{}
}
