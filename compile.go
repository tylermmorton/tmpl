package tmpl

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"os"
	"reflect"
)

// TemplateTextProvider is a type that provides the raw text of a tmpl file. This
// is usually used in conjunction with an embedded template.
type TemplateTextProvider interface {
	TemplateText() string
}

// assertTextProvider asserts that the given value or pointer to the value is
// an instance of TemplateTextProvider
func assertTextProvider(val *reflect.Value) (TemplateTextProvider, bool) {
	if p, ok := val.Interface().(TemplateTextProvider); ok {
		return p, true
		//} else if p, ok := val.Addr().Interface().(TemplateTextProvider); ok {
		//	return p, true
	}
	return nil, false
}

// TemplateFileProvider is a type that provides an absolute path to a tmpl file. This
// is usually used in conjunction with a watched template.
type TemplateFileProvider interface {
	TemplateFile() string
}

// assertFileProvider asserts that the given value or pointer to the value is
// an instance of TemplateFileProvider
func assertFileProvider(val *reflect.Value) (TemplateFileProvider, bool) {
	if p, ok := val.Interface().(TemplateFileProvider); ok {
		return p, true
		//} else if p, ok := val.Addr().Interface().(TemplateFileProvider); ok {
		//	return p, true
	} else {
		return nil, false
	}
}

// compile recursively compiles all fields into a single Template instance.
func compile(ctx context.Context, p Parser, val reflect.Value, t *template.Template) (*template.Template, error) {
	var wrapWithDefinition = func(name string, body string) string {
		return fmt.Sprintf("{{define %q -}}\n%s{{end}}\n", name, body)
	}

	for i := 0; i < val.NumField(); i++ {
		name, ok := val.Type().Field(i).Tag.Lookup("tmpl")
		if !ok {
			name = val.Type().Field(i).Name
		}

		var templateText string
		var field = val.Field(i)
		if provider, ok := assertTextProvider(&field); ok {
			templateText = provider.TemplateText()
		} else if provider, ok := assertFileProvider(&field); ok {
			byt, err := os.ReadFile(provider.TemplateFile())
			if err != nil {
				panic(err)
			}
			templateText = string(byt)
		} else {
			// TODO: refactor this...
			continue
		}

		t = template.Must(p.Parse(t, name, wrapWithDefinition(name, templateText)))
		t = template.Must(compile(ctx, p, reflect.ValueOf(field).Elem(), t))
	}

	return t, nil
}

// TemplateProvider is an alias for a type that implements either
// the TemplateTextProvider or TemplateFileProvider interfaces.
//
// TODO: If Go ever adds union types on behavioral interfaces...
type TemplateProvider = interface {
	// TemplateTextProvider | TemplateFileProvider
}

func Compile[T TemplateProvider](tp T) Template[T] {
	var ctx = context.Background()
	var p = NewParser()
	var t = template.New(fmt.Sprintf("%T", tp))

	log.Printf("Compiling Template[%T]\n", tp)

	var templateText string
	var val = reflect.ValueOf(tp)
	if provider, ok := assertTextProvider(&val); ok {
		templateText = provider.TemplateText()
	} else if provider, ok := assertFileProvider(&val); ok {
		byt, err := os.ReadFile(provider.TemplateFile())
		if err != nil {
			panic(err)
		}
		templateText = string(byt)
	} else {
		panic(fmt.Sprintf("Expected type %T to implement TemplateFileProvider or TemplateTextProvider but it does not implement either.", provider))
	}
	t = template.Must(p.Parse(t, fmt.Sprintf("%T", tp), templateText))
	t = template.Must(compile(ctx, p, reflect.ValueOf(tp).Elem(), t))

	return &renderer[T]{
		ctx:      ctx,
		template: t,
	}
}
