package tmpl

import (
	"errors"
	"fmt"
	"html/template"
	"io"
)

type RenderProcess struct {
	Target   string
	Template *template.Template
}

type RenderOption func(p *RenderProcess)

// WithName copies the Template's default parse.Tree and adds it back
// to the Template under the given name, effectively aliasing the Template.
func WithName(name string) RenderOption {
	return func(p *RenderProcess) {
		template.Must(p.Template.AddParseTree(name, p.Template.Tree.Copy()))
	}
}

// WithTarget sets the render Target to the given Template name.
func WithTarget(target string) RenderOption {
	return func(p *RenderProcess) {
		p.Target = target
	}
}

// WithFuncs appends the given Template.FuncMap to the Template's internal
// func map. These functions become available in the Template during execution
func WithFuncs(funcs template.FuncMap) RenderOption {
	return func(p *RenderProcess) {
		p.Template = p.Template.Funcs(funcs)
	}
}

func (tmpl *tmpl[T]) Render(wr io.Writer, data T, opts ...RenderOption) error {
	tmpl.mu.RLock()
	t, err := tmpl.template.Clone()
	tmpl.mu.RUnlock()
	if err != nil {
		return err
	}

	// defer a panic boundary to catch errors thrown by any of the
	// given visitor functions
	defer func() {
		if r := recover(); r != nil {
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = fmt.Errorf("recovered panic during Template render option: %v", t)
			}
		}
	}()

	p := &RenderProcess{
		Template: t,
		Target:   t.Tree.ParseName,
	}

	for _, opt := range opts {
		opt(p)
	}

	return t.ExecuteTemplate(wr, p.Target, data)
}
