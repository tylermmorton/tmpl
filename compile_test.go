package tmpl

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"
)

type Component struct {
	Text string
}

func (*Component) TemplateText() string {
	return "{{.Text}}"
}

//go:embed compile_test.tmpl.html
var testTemplateText string

type TestTemplate struct {
	Message Component `tmpl:"message"`
	// Components tests slices of nested templates
	Components []Component `tmpl:"component"`

	Name string
}

func (*TestTemplate) TemplateText() string {
	return testTemplateText
}

func Test_Compile(t *testing.T) {
	tmpl, err := Compile(&TestTemplate{})
	if err != nil {
		t.Fatal(err)
	}

	buf := bytes.Buffer{}
	err = tmpl.Render(&buf, &TestTemplate{
		Name: "World",
		Message: Component{
			Text: "Hello",
		},
		Components: []Component{
			{
				Text: "Thank you for ",
			},
			{
				Text: "trying out tmpl",
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(buf.String(), "Hello World") {
		t.Logf("output: %s", buf.String())
		t.Fatal("expected to find 'Hello World' in the output")
	}

	if !strings.Contains(buf.String(), "Thank you for trying out tmpl") {
		t.Logf("output: %s", buf.String())
		t.Fatal("expected to find 'Thank you for trying out tmpl' in the output")
	}
}
