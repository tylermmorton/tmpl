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

//func (tmpl *tmpl[T]) spawnWatcherRoutine(w Watcher) {
//	ch := make(chan struct{})
//	ec := make(chan error)
//
//	// spawn a goroutine to listen for new template files on the channel
//	go func() {
//		for {
//			select {
//			case <-ch:
//				err := tmpl.compile(template.New(fmt.Sprintf("%T", tmpl.provider)), reflect.ValueOf(tmpl.provider).Elem())
//				if err != nil {
//					log.Printf("[Watch] Failed to compile %T: %+v\n", w, err)
//				}
//			case err := <-ec:
//				log.Printf("[Watch] Failed to watch %T: %+v\n", w, err)
//			}
//		}
//	}()
//
//	// register the channels with the object in charge of
//	// watching the template file
//	w.WatchSignal(ch, ec)
//}

// Compile takes the given TemplateProvider, parses the template text and then
// recursively compiles all nested templates into one managed Template instance.
func Compile[T TemplateProvider](p T) (Template[T], error) {
	var (
		n = strings.TrimPrefix(fmt.Sprintf("%T", p), "*")
		c = &compiler{
			ctx:    context.Background(),
			parser: NewParser(),
		}
	)

	t, err := c.compile(nil, n, p)
	if err != nil {
		return nil, fmt.Errorf("failed to compile template: %w", err)
	}

	return &tmpl[T]{
		ctx:      c.ctx,
		mu:       &sync.RWMutex{},
		name:     t.Name(),
		template: t,
	}, nil
}

func MustCompile[T TemplateProvider](p T) Template[T] {
	tmpl, err := Compile(p)
	if err != nil {
		panic(err)
	}
	return tmpl
}
