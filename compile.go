package tmpl

import (
	"fmt"
	"html/template"
	"reflect"
	"sync"
)

// CompilerOptions holds options that control the template compiler
type CompilerOptions struct {
	// analyzers is a list of analyzers that are run before compilation
	analyzers []Analyzer
	// parseOpts are the options passed to the template parser
	parseOpts ParseOptions
}

// CompilerOption is a function that can be used to modify the CompilerOptions
type CompilerOption func(opts *CompilerOptions)

func UseAnalyzers(analyzers ...Analyzer) CompilerOption {
	return func(opts *CompilerOptions) {
		opts.analyzers = append(opts.analyzers, analyzers...)
	}
}

// UseParseOptions sets the ParseOptions for the template CompilerOptions. These
// options are used internally with the html/template package.
func UseParseOptions(parseOpts ParseOptions) CompilerOption {
	return func(opts *CompilerOptions) {
		opts.parseOpts = parseOpts
	}
}

func compile(tp TemplateProvider, opts ParseOptions, analyzers ...Analyzer) (*template.Template, error) {
	var (
		err error
		t   *template.Template
	)

	helper, err := Analyze(tp, opts, analyzers)
	if err != nil {
		return nil, err
	}

	// recursively parse all templates into a single template instance
	// this block is responsible for constructing the template that
	// will be rendered by the user
	err = recurseFieldsImplementing[TemplateProvider](tp, func(tp TemplateProvider, field reflect.StructField) error {
		var templateText string

		templateName, ok := field.Tag.Lookup("tmpl")
		if !ok {
			templateName = field.Name
		}

		if t == nil {
			// if t is nil, that means this is the recursive entrypoint
			// and some construction needs to happen
			t = template.New(templateName)
			templateText = tp.TemplateText()

			t = t.Delims(opts.LeftDelim, opts.RightDelim)

			// Analyzers can provide functions to be used in templates
			t = t.Funcs(helper.FuncMap())
		} else {
			// if this is a nested template wrap its text in a {{ define }}
			// statement, so it may be referenced by the "parent" template
			// ex: {{define %q -}}\n%s{{end}}
			templateText = fmt.Sprintf("%[1]sdefine %[3]q -%[2]s\n%[4]s%[1]send%[2]s\n", opts.LeftDelim, opts.RightDelim, templateName, tp.TemplateText())
		}

		t, err = t.Parse(templateText)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to compile template: %+v", err)
	}

	return t, nil
}

// Compile takes the given TemplateProvider, parses the templateProvider text and then
// recursively compiles all nested templates into one managed Template instance.
//
// Compile also spawns a watcher routine. If the given TemplateProvider or any
// nested templates within implement TemplateWatcher, they can send signals over
// the given channel when it is time for the templateProvider to be recompiled.
func Compile[T TemplateProvider](tp T, opts ...CompilerOption) (Template[T], error) {
	var (
		c = &CompilerOptions{
			analyzers: builtinAnalyzers,
			parseOpts: ParseOptions{
				LeftDelim:  "{{",
				RightDelim: "}}",
			},
		}
	)

	for _, opt := range opts {
		opt(c)
	}

	m := &managedTemplate[T]{
		mu: &sync.RWMutex{},
	}

	doCompile := func() error {
		t, err := compile(tp, c.parseOpts, c.analyzers...)
		if err != nil {
			return err
		}

		m.mu.Lock()
		m.template = t
		m.mu.Unlock()

		return nil
	}

	err := doCompile()
	if err != nil {
		return nil, err
	}

	// recursively spawn goroutines to watch for recompile signals
	_ = recurseFieldsImplementing[TemplateWatcher](tp, func(w TemplateWatcher, field reflect.StructField) (err error) {
		go w.Watch(doCompile)
		return
	})

	return m, nil
}

func MustCompile[T TemplateProvider](p T, opts ...CompilerOption) Template[T] {
	tmpl, err := Compile(p, opts...)
	if err != nil {
		panic(err)
	}
	return tmpl
}
