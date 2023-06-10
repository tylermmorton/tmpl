package tmpl

import (
	"fmt"
	"html/template"
	"reflect"
	"sync"
)

// compiler is the internal compiler instance
type compiler struct {
	// parseOpts are the options passed to the templateProvider parser
	parseOpts ParseOptions
	// analyzers is a list of analyzers that are run on the templateProvider
	analyzers []Analyzer
	// signal is the signal channel that recompiles the templateProvider
	signal chan struct{}
}

// CompilerOption is a function that can be used to modify the compiler
type CompilerOption func(c *compiler)

func UseAnalyzers(analyzers ...Analyzer) CompilerOption {
	return func(c *compiler) {
		c.analyzers = append(c.analyzers, analyzers...)
	}
}

// UseParseOptions sets the ParseOptions for the templateProvider compiler. These
// options are used internally with the html/templateProvider package.
func UseParseOptions(opts ParseOptions) CompilerOption {
	return func(c *compiler) {
		c.parseOpts = opts
	}
}

func compile(tp TemplateProvider, opts ParseOptions) (*template.Template, error) {
	var (
		err error
		t   *template.Template
	)

	reporter, err := Analyze(tp, opts, builtinAnalyzers)
	if err != nil {
		return nil, err
	}

	// recursively parse all templates into a single templateProvider instance
	// this block is responsible for constructing the templateProvider that
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
			t = t.Funcs(reporter.FuncMap())
		} else {
			// if this is a nested templateProvider wrap its text in a {{ define }}
			// statement, so it may be referenced by the "parent" templateProvider
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
		return nil, fmt.Errorf("failed to compile templateProvider: %+v", err)
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
		c = &compiler{
			analyzers: builtinAnalyzers,
			signal:    make(chan struct{}),
			parseOpts: ParseOptions{
				LeftDelim:  "{{",
				RightDelim: "}}",
			},
		}
	)

	for _, opt := range opts {
		opt(c)
	}

	t, err := compile(tp, c.parseOpts)
	if err != nil {
		return nil, err
	}

	// recursively spawn goroutines to watch for recompile signals
	err = recurseFieldsImplementing[TemplateWatcher](tp, func(w TemplateWatcher, field reflect.StructField) error {
		go w.Watch(c.signal)
		return nil
	})

	return &tmpl[T]{
		mu:       &sync.RWMutex{},
		template: t,
	}, nil
}

func MustCompile[T TemplateProvider](p T, opts ...CompilerOption) Template[T] {
	tmpl, err := Compile(p, opts...)
	if err != nil {
		panic(err)
	}
	return tmpl
}
