package tmpl

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"io"
)

type RenderProcess struct {
	Targets  []string
	Template *template.Template
}

type RenderOption func(p *RenderProcess)

// WithName copies the Template's default parse.Tree and adds it back
// to the Template under the given name, effectively aliasing the Template.
func WithName(name string) RenderOption {
	return func(p *RenderProcess) {
		p.Template = template.Must(p.Template.AddParseTree(name, p.Template.Tree.Copy()))
	}
}

// WithTarget sets the render Target to the given Template name.
func WithTarget(target ...string) RenderOption {
	return func(p *RenderProcess) {
		p.Targets = append(p.Targets, target...)
	}
}

// WithFuncs appends the given Template.FuncMap to the Template's internal
// func map. These functions become available in the Template during execution
func WithFuncs(funcs template.FuncMap) RenderOption {
	return func(p *RenderProcess) {
		p.Template = p.Template.Funcs(funcs)
	}
}

func (tmpl *managedTemplate[T]) Render(wr io.Writer, data T, opts ...RenderOption) error {
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
		Targets:  []string{},
	}

	for _, opt := range opts {
		opt(p)
	}

	// render the default template if no targets are provided.
	if len(p.Targets) == 0 {
		p.Targets = append(p.Targets, t.Tree.ParseName)
	}

	buf := bytes.Buffer{}
	for _, target := range p.Targets {
		if err := p.Template.ExecuteTemplate(&buf, target, data); err != nil {
			return err
		}
	}

	_, err = wr.Write(buf.Bytes())
	return err
}

func (tmpl *managedTemplate[T]) RenderToChan(ch chan string, data T, opts ...RenderOption) error {
	buf := bytes.Buffer{}
	err := tmpl.Render(&buf, data, opts...)
	if err != nil {
		return err
	}
	ch <- buf.String()
	return nil
}

func (tmpl *managedTemplate[T]) RenderToString(data T, opts ...RenderOption) (string, error) {
	buf := bytes.Buffer{}
	err := tmpl.Render(&buf, data, opts...)
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}
