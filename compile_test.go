package tmpl

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"
)

// TODO: refactor these tp structs to use tmpl:bind

type TextComponent struct {
	Text string
}

func (*TextComponent) TemplateText() string {
	return "{{.Text}}"
}

type ScriptComponent struct {
	Source string
}

func (*ScriptComponent) TemplateText() string {
	return "<script src=\"{{.Source}}\"></script>"
}

type UndefinedTemplate struct{}

func (*UndefinedTemplate) TemplateText() string {
	return `{{ template "undefined" }}`
}

type UndefinedRange struct{}

func (*UndefinedRange) TemplateText() string {
	return `{{ range .List }}{{ end }}`
}

type TestTemplate struct {
	// Name tests fields who do not implement TemplateProvider
	Name string
	// Content tests unnamed TemplateProviders
	Content *TextComponent
	// Title tests struct values
	Title string
	// Components tests slices of nested templates
	Scripts []ScriptComponent `tmpl:"script"`
}

//go:embed testdata/compiler_test.tmpl.html
var testTemplateText string

func (*TestTemplate) TemplateText() string {
	return testTemplateText
}

// Test_Compile tests the compiler's ability to compile and render templates.
// It's like a package level integration test at this point
func Test_Compile(t *testing.T) {
	testCases := map[string]struct {
		tp TemplateProvider

		expectRenderOutput []string
		expectRenderErrMsg string

		expectCompileErrMsg string
	}{
		// tmpl should support all html/template syntax. these test cases are
		// to ensure the compiler is not breaking any of the syntax. for sanity
		"Supports usage of {{ . }} pipeline statements": {
			tp:                 &TextComponent{Text: "Hello World"},
			expectRenderOutput: []string{"Hello World"},
		},
		"Supports usage of {{ define }} and {{ template }} statements": {
			tp: &TestTemplate{
				Title:   "Test",
				Scripts: []ScriptComponent{},
				Content: &TextComponent{Text: "Hello World"},
			},
			expectRenderOutput: []string{
				"Test",
				"Hello World",
				"<form id=\"defineForm\">",
			},
		},
		"Supports usage of {{ range }} statements": {
			tp: &TestTemplate{
				Title: "Test",
				Scripts: []ScriptComponent{
					{Source: "script1.js"},
					{Source: "script2.js"},
				},
				Content: &TextComponent{Text: "Hello World"},
			},
			expectRenderOutput: []string{
				"Test",
				"Hello World",
				"<script src=\"script1.js\"></script>",
				"<script src=\"script2.js\"></script>",
			},
		},

		// these are test cases for the compiler's built-in analyzers
		"Catches usage of {{ template }} statements containing undefined template names": {
			tp:                  &UndefinedTemplate{},
			expectCompileErrMsg: "template \"undefined\" is not provided",
		},
		"Catches usage of {{ range }} statements containing undefined fields": {
			tp:                  &UndefinedRange{},
			expectCompileErrMsg: "field \"List\" not defined",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tmpl, err := Compile(tc.tp)
			if err != nil {
				if len(tc.expectCompileErrMsg) == 0 {
					t.Fatal(err)
				} else if !strings.Contains(err.Error(), tc.expectCompileErrMsg) {
					t.Fatalf("expected compile error message to contain %q, got %q", tc.expectCompileErrMsg, err.Error())
				} else {
					return
				}
			}

			buf := bytes.Buffer{}
			err = tmpl.Render(&buf, tc.tp)
			if err != nil {
				if len(tc.expectRenderErrMsg) == 0 {
					t.Fatal(err)
				} else if !strings.Contains(err.Error(), tc.expectRenderErrMsg) {
					t.Fatalf("expected render error message to contain %q, got %q", tc.expectRenderErrMsg, err.Error())
				} else {
					return
				}
			}

			for _, expect := range tc.expectRenderOutput {
				if !strings.Contains(buf.String(), expect) {
					t.Fatalf("expected render output to contain %q, got %q", expect, buf.String())
				}
			}
		})
	}
}
