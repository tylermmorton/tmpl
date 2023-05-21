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
	// Name tests fields who do not implement TemplateProvider
	Name string
	// Content tests unnamed TemplateProviders
	Content Component
	// Title tests struct values
	Title Component `tmpl:"title"`
	// Greeting tests pointers to struct values
	Greeting *Component `tmpl:"greeting"`
	// Components tests slices of nested templates
	Components []*Component `tmpl:"component"`
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
		Name:     "World",
		Title:    Component{Text: "tmpl | test suite"},
		Greeting: &Component{Text: "Hello"},
		Content:  Component{Text: "Go templates are cool"},
		Components: []*Component{
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

	if !strings.Contains(buf.String(), "Go templates are cool") {
		t.Logf("output: %s", buf.String())
		t.Fatal("expected to find 'Go templates are cool' in the output")
	}

	if !strings.Contains(buf.String(), "tmpl | test suite") {
		t.Logf("output: %s", buf.String())
		t.Fatal("expected to find 'tmpl | test suite' in the output")
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
