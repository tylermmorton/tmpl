package tmpl

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"reflect"
	"sync"
)

func (tmpl *tmpl[T]) compile() error {
	var (
		err      error
		name     = fmt.Sprintf("%T", tmpl.p)
		provider = tmpl.p
		t        = template.New(name)
		val      = reflect.ValueOf(provider).Elem()
	)

	t, err = tmpl.parser.Parse(t, name, provider.TemplateText())
	if err != nil {
		return err
	}

	t, err = tmpl.compileNested(t, val)
	if err != nil {
		return err
	}

	tmpl.mu.Lock()
	tmpl.template = t
	tmpl.mu.Unlock()

	return nil
}

// compileNested recursively compiles all struct fields who implement TemplateProvider
// into the given template instance.
func (tmpl *tmpl[T]) compileNested(t *template.Template, val reflect.Value) (*template.Template, error) {
	define := func(name string, body string) string {
		return fmt.Sprintf("{{define %q -}}\n%s{{end}}\n", name, body)
	}
	doCompile := func(t *template.Template, name string, tp TemplateProvider) (err error) {
		t, err = tmpl.parser.Parse(t, name, define(name, tp.TemplateText()))
		if err != nil {
			return err
		}

		t, err = tmpl.compileNested(t, reflect.ValueOf(tp).Elem())
		if err != nil {
			return err
		}

		return
	}

	for i := 0; i < val.NumField(); i++ {
		name, ok := val.Type().Field(i).Tag.Lookup("tmpl")
		if !ok {
			name = val.Type().Field(i).Name
		}

		if tp, ok := val.Field(i).Interface().(TemplateProvider); ok {
			err := doCompile(t, name, tp)
			if err != nil {
				return nil, err
			}
		} else if tp, ok := val.Field(i).Addr().Interface().(TemplateProvider); ok {
			err := doCompile(t, name, tp)
			if err != nil {
				return nil, err
			}
		}
	}

	return t, nil
}

func (tmpl *tmpl[T]) spawnWatcherRoutine(w Watcher) {
	ch := make(chan struct{})
	ec := make(chan error)

	// spawn a goroutine to listen for new template files on the channel
	go func() {
		for {
			select {
			case <-ch:
				err := tmpl.compile()
				if err != nil {
					log.Printf("[Watch] Failed to compile %T: %+v\n", w, err)
				}
			case err := <-ec:
				log.Printf("[Watch] Failed to watch %T: %+v\n", w, err)
			}
		}
	}()

	// register the channels with the object in charge of
	// watching the template file
	w.WatchSignal(ch, ec)
}

func Compile[T TemplateProvider](p T) (Template[T], error) {
	c := &tmpl[T]{
		ctx:      context.Background(),
		mu:       &sync.RWMutex{},
		parser:   NewParser(),
		p:        p,
		template: template.New(fmt.Sprintf("%T", p)),
	}

	err := c.compile()
	if err != nil {
		return nil, fmt.Errorf("failed to compile template: %w", err)
	}

	if w, ok := reflect.ValueOf(p).Interface().(Watcher); ok {
		c.spawnWatcherRoutine(w)
	}

	return c, nil
}

func MustCompile[T TemplateProvider](p T) Template[T] {
	tmpl, err := Compile(p)
	if err != nil {
		panic(err)
	}
	return tmpl
}
