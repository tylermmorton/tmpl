package tmpl

import (
	"bytes"
	"github.com/fsnotify/fsnotify"
	"testing"
)

type TestTextProvider struct {
	Name string
}

func (ct *TestTextProvider) TemplateText() string {
	return "Hello, {{.Name}}!"
}

func (ct *TestTextProvider) WatchSignal(signal chan struct{}, ch chan error) {
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

	// Add a path.
	err = watcher.Add("/abs/path/app.tmpl.html")
	if err != nil {
		ch <- err
		return
	}

	// Block goroutine forever so the watcher doesn't get gc'd
	<-make(chan struct{})
}

type TestNestedTemplate struct {
	TestTextProvider `tmpl:"nested"`
}

func (nt *TestNestedTemplate) TemplateText() string {
	return `{{ template "nested" . }}`
}

func Test_Compile(t *testing.T) {
	testTable := map[string]struct {
		provider   TemplateProvider
		expected   string
		wantErr    bool
		wantErrMsg string
	}{
		"Can compileNested a TextProvider passed by reference": {
			provider: &TestTextProvider{
				Name: "World",
			},
			expected: "Hello, World!",
		},

		"Can compileNested a nested TextProvider": {
			provider: &TestNestedTemplate{
				TestTextProvider: TestTextProvider{
					Name: "World",
				},
			},
			expected: "Hello, World!",
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			buf := bytes.Buffer{}
			err := MustCompile(test.provider).Render(&buf, &TestTextProvider{Name: "World"})
			if err != nil {
				if !test.wantErr {
					t.Errorf("Compilation failed: %+v", err)
				}
				if test.wantErrMsg != "" && err.Error() != test.wantErrMsg {
					t.Errorf("Expected error message %q, got %q", test.wantErrMsg, err.Error())
				}
			} else {
				if test.wantErr {
					t.Errorf("Expected error, got none")
				}
			}
			if buf.String() != test.expected {
				t.Errorf("Expected %q, got %q", test.expected, buf.String())
			}
		})
	}
}
