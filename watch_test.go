package tmpl

import (
	"bytes"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"
)

const testRenders = 10
const testTemplateFile = "./testdata/templates/watch_test.tmpl.html"

func writeTemplateFile(t *testing.T, text string) {
	err := os.WriteFile(testTemplateFile, []byte(text), 0644)
	if err != nil {
		t.Fatalf("failed to write to testTemplateFile: %+v", err)
	}
}

//go:generate tmpl bind ./watch_test.go --outfile=watch_gen_test.go
//tmpl:bind ./testdata/templates/watch_test.tmpl.html
type watcherTestTemplate struct{}

func (t *watcherTestTemplate) Watch(compile func() error) {
	go func() {
		for i := 0; i <= testRenders; i++ {
			err := compile()
			if err != nil {
				panic(err)
			}

			err = os.WriteFile(testTemplateFile, []byte(fmt.Sprintf("%d", i)), 0644)
			if err != nil {
				panic(err)
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()
}

var _ interface {
	TemplateProvider
	TemplateWatcher
} = (*watcherTestTemplate)(nil)

func Test_Watch(t *testing.T) {
	writeTemplateFile(t, ``)

	template, err := Compile(&watcherTestTemplate{})
	if err != nil {
		t.Fatalf("failed to compile template: %+v", err)
	}

	wg := &sync.WaitGroup{}
	for i := 0; i < testRenders; i++ {
		go func(i int, wg *sync.WaitGroup) {
			wg.Add(1)
			defer wg.Done()

			for {
				buf := bytes.Buffer{}
				err = template.Render(&buf, &watcherTestTemplate{})
				if err != nil {
					panic(err)
				}

				if buf.String() == fmt.Sprintf("%d", i) {
					break
				}

				time.Sleep(10 * time.Millisecond)
			}
		}(i, wg)
	}

	wg.Wait()
}
