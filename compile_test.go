package tmpl

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"
)

// TODO: refactor these templateProvider structs to use tmpl:bind

type TextComponent struct {
	Text string
}

func (t *TextComponent) String() string {
	return t.Text
}

func (*TextComponent) TemplateText() string {
	return "{{.}}"
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
	return `{{ range .UndList }}{{ end }}`
}

type DefinedField struct {
	DefField string
}

func (*DefinedField) TemplateText() string {
	return `{{ .DefField }}`
}

type DefinedNestedField struct {
	Nested DefinedField
}

func (*DefinedNestedField) TemplateText() string {
	return `{{ .Nested.DefField }}`
}

type UndefinedField struct{}

func (*UndefinedField) TemplateText() string {
	return `{{ .UndField }}`
}

type UndefinedNestedField struct {
	Nested UndefinedField
}

func (*UndefinedNestedField) TemplateText() string {
	return `{{ .Nested.UndField }}`
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
		templateProvider TemplateProvider

		expectRenderOutput []string
		expectRenderErrMsg string

		expectCompileErrMsg string
	}{
		// tmpl should support all html/template syntax. these test cases are
		// to ensure the compiler is not breaking any of the syntax. for sanity
		"Supports usage of {{ . }} pipeline statements": {
			templateProvider:   &TextComponent{Text: "Hello World"},
			expectRenderOutput: []string{"Hello World"},
		},
		"Supports usage of {{ .Field }} pipeline statements": {
			templateProvider:   &DefinedField{DefField: "Hello World"},
			expectRenderOutput: []string{"Hello World"},
		},
		"Supports usage of {{ .Nested.Field }} pipeline statements": {
			templateProvider:   &DefinedNestedField{Nested: DefinedField{DefField: "Hello World"}},
			expectRenderOutput: []string{"Hello World"},
		},
		"Supports usage of {{ define }} and {{ template }} statements": {
			templateProvider: &TestTemplate{
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
			templateProvider: &TestTemplate{
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
			templateProvider:    &UndefinedTemplate{},
			expectCompileErrMsg: "template \"undefined\" is not provided",
		},
		"Catches usage of {{ range }} statements containing undefined fields": {
			templateProvider:    &UndefinedRange{},
			expectCompileErrMsg: "field \".UndList\" not defined",
		},
		"Catches usage of undefined fields": {
			templateProvider:    &UndefinedField{},
			expectCompileErrMsg: "field \".UndField\" not defined",
		},
		"Catches usage of undefined nested fields": {
			templateProvider:    &UndefinedNestedField{Nested: UndefinedField{}},
			expectCompileErrMsg: "field \".Nested.UndField\" not defined",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			tmpl, err := Compile(tc.templateProvider)
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
			err = tmpl.Render(&buf, tc.templateProvider)
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
