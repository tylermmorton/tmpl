package testdata

import (
	"github.com/fsnotify/fsnotify"
	"os"
)

func (t *TestTemplate) TemplateText() string {
	byt, err := os.ReadFile("/Users/tylermorton/go/src/github.com/tylermmorton/tmpl/cmd/tmpl/testdata/test.tmpl.html")
	if err != nil {
		panic(err)
	}
	return string(byt)
}

func (t *Test2Template) TemplateText() string {
	byt, err := os.ReadFile("/Users/tylermorton/go/src/github.com/tylermmorton/tmpl/cmd/tmpl/testdata/test2.tmpl.html")
	if err != nil {
		panic(err)
	}
	return string(byt)
}

func (t *Test2Template) WatchSignal(signal chan struct{}, ch chan error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		ch <- err
		return
	}
	defer watcher.Close()

	// Recover any panics from reading the file
	// and pass it along the given error channel
	defer func(ch chan error) {
		if err, ok := recover().(error); ok && err != nil {
			ch <- err
		}
	}(ch)

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Write) {
					signal <- struct{}{}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				ch <- err
			}
		}
	}()

	err = watcher.Add("/Users/tylermorton/go/src/github.com/tylermmorton/tmpl/cmd/tmpl/testdata/test2.tmpl.html")
	if err != nil {
		ch <- err
		return
	}

	// Block goroutine forever so the watcher doesn't get gc'd
	<-make(chan struct{})
}
