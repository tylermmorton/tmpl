package tmpl

import (
	"context"
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"sync"
)

// compiler is the internal compiler instance
type compiler struct {
	ctx    context.Context
	parser Parser
	// signal is the signal channel that recompiles the template
	signal chan struct{}
}

func wrapTreeDefinition(name string, body string) string {
	return fmt.Sprintf("{{define %q -}}\n%s{{end}}\n", name, body)
}

// compile is the template compiler implementation. it orchestrates the entire compilation process.
func (c *compiler) compile(t *template.Template, templateName string, p TemplateProvider) (*template.Template, error) {
	var (
		err          error
		ok           bool
		templateText string
	)

	if t == nil {
		// if t is nil, that means this is the recursive entrypoint
		t = template.New(templateName)
		templateText = p.TemplateText()
	} else {
		templateText = wrapTreeDefinition(templateName, p.TemplateText())
	}

	// TODO(tylermorton): I'd like to use tmpl.Parser directly here,
	//  but its not possible as you cannot add multiple parse.Trees
	//  to a template.[1] You must call t.Parse multiple times instead
	//  Is this a bug in the html/template package? :/
	//    - [1] why? bc t.AddParseTree overwrites the previously added
	//          tree but t.Parse appends it ??
	t, err = t.Parse(templateText)
	if err != nil {
		return nil, err
	}

	// Recursively compile any fields of this TemplateProvider
	//   Fields can be:
	//     - Struct value
	//     - Pointer to struct
	//     - Slice of struct values or pointers to structs
	//     - Pointer to slice of struct values or pointers to structs :)
	var val = reflect.ValueOf(p).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		if field.Kind() != reflect.Ptr &&
			field.Kind() != reflect.Slice &&
			field.Kind() != reflect.Struct {
			continue
		}

		// templates can be named via the `tmpl` struct tag.
		// otherwise they get named by their field name
		templateName, ok = val.Type().Field(i).Tag.Lookup("tmpl")
		if !ok {
			templateName = val.Type().Field(i).Name
		}

		var intf reflect.Value
		switch field.Kind() {
		case reflect.Struct:
			if field.Type().Kind() == reflect.Ptr {
				intf = reflect.New(field.Type().Elem())
			} else {
				intf = reflect.New(field.Type())
			}
		case reflect.Ptr:
			fallthrough
		case reflect.Slice:
			if field.Type().Elem().Kind() == reflect.Ptr {
				intf = reflect.New(field.Type().Elem().Elem())
			} else {
				intf = reflect.New(field.Type().Elem())
			}
		}

		// assert if the value implements TemplateWatcher
		if w, ok := intf.Interface().(TemplateWatcher); ok {
			go w.Watch(c.signal)
		}

		// assert if the value implements TemplateProvider
		if tp, ok := intf.Interface().(TemplateProvider); ok {
			t, err = c.compile(t, templateName, tp)
			if err != nil {
				return nil, err
			}
		}

	}

	return t, nil
}

func (c *compiler) watch(t *template.Template, mu *sync.RWMutex, relay chan struct{}, p TemplateProvider) {
	var (
		n = strings.TrimPrefix(fmt.Sprintf("%T", p), "*")
	)

	for {
		select {
		case <-c.signal:
			mu.Lock()
			temp, err := c.compile(nil, n, p)
			if err != nil {
				fmt.Printf("[watch] build failed: %+v", err)
			} else {
				// overwrite the internal template pointer and
				// notify any listeners of a successful recompile
				t = temp
				relay <- struct{}{}
			}
			mu.Unlock()
		}
	}
}

// Compile takes the given TemplateProvider, parses the template text and then
// recursively compiles all nested templates into one managed Template instance.
//
// Compile also spawns a watcher routine. If the given TemplateProvider or any
// nested templates within implement TemplateWatcher, they can send signals over
// the given channel when it is time for the template to be recompiled.
func Compile[T TemplateProvider](p T) (Template[T], error) {
	var (
		n = strings.TrimPrefix(fmt.Sprintf("%T", p), "*")
		c = &compiler{
			ctx:    context.Background(),
			parser: NewParser(),
			signal: make(chan struct{}),
		}
		mu = &sync.RWMutex{}
	)

	t, err := c.compile(nil, n, p)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template: %w", err)
	}

	// relay represents an additional channel to receive
	// signals when the compiler completes a watch build
	// TODO: there's currently no api that can leverage this
	relay := make(chan struct{})

	// spawn a thread to receive recompile signals
	// from any templates who implement TemplateWatcher
	go c.watch(t, mu, relay, p)

	return &tmpl[T]{
		ctx:      c.ctx,
		mu:       mu,
		name:     t.Name(),
		template: t,
		signal:   relay,
	}, nil
}

func MustCompile[T TemplateProvider](p T) Template[T] {
	tmpl, err := Compile(p)
	if err != nil {
		panic(err)
	}
	return tmpl
}
