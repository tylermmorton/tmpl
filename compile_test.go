package tmpl

import (
	"bytes"
	_ "embed"
	"strings"
	"testing"

	. "github.com/tylermmorton/tmpl/testdata"
)

// TODO: replace tests with table driven tests
// @deprecated
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
		renderOptions    []RenderOption

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
		"Supports usage of {{ if }} statements with bare fields": {
			templateProvider: &DefinedIf{DefIf: true, Message: "Hello World"},
			expectRenderOutput: []string{
				"Hello World",
			},
		},
		"Supports usage of builtin equality operations in {{ if eq .Field 1 }} pipelines": {
			templateProvider: &PipelineIf{
				DefInt:  1,
				Message: "Hello World",
			},
			expectRenderOutput: []string{
				"Hello World",
			},
		},
		"Supports usage of {{ range }} statements over string types": {
			templateProvider: &DefinedRange{DefList: []string{"Hello", "World"}},
			expectRenderOutput: []string{
				"Hello",
				"World",
			},
		},
		"Supports usage of {{ range }} statements over anonymous struct types": {
			templateProvider: &StructRange{DefList: []struct {
				DefField string
			}{
				{DefField: "Hello"},
				{DefField: "World"},
			}},
			expectRenderOutput: []string{
				"Hello",
				"World",
			},
		},
		"Supports usage of {{ range }} statements over named struct types": {
			templateProvider: &NamedStructRange{NamedStructs: []NamedStruct{
				{DefField: "Hello"},
				{DefField: "World"},
			}},
			expectRenderOutput: []string{
				"Hello",
				"World",
			},
		},

		"Supports usage of {{ if }} statements within {{ range }} bodies": {
			templateProvider: &IfWithinRange{
				DefList: []DefinedIf{
					{DefIf: true, Message: "Hello"},
				},
			},
			expectRenderOutput: []string{
				"Hello",
			},
		},
		"Supports usage of {{ range }} statements within {{ range }} bodies": {
			templateProvider: &StructRangeWithinRange{
				ListOne: []StructOne{
					{
						ListTwo: []StructTwo{
							{DefField: "Hello"},
						},
					},
					{
						ListTwo: []StructTwo{
							{DefField: "World"},
						},
					},
				},
			},
			expectRenderOutput: []string{
				"Hello",
				"World",
			},
		},
		// template nesting tests
		"Supports embedded struct fields": {
			templateProvider: &EmbeddedField{
				EmbeddedStruct: EmbeddedStruct{DefField: "Hello World"},
			},
			expectRenderOutput: []string{"Hello World"},
		},
		"Supports multiple levels of embedded TemplateProviders": {
			templateProvider: &MultiLevelEmbeds{
				LevelOneEmbed: LevelOneEmbed{
					LevelTwoEmbed: LevelTwoEmbed{
						DefField: "Hello World",
					},
				},
			},
			expectRenderOutput: []string{"Hello World"},
		},

		// layout & outlet tests (RenderOption tests)
		"Supports usage of WithTarget and WithName when rendering templates": {
			templateProvider: &Outlet{
				Layout:  Layout{},
				Content: "Hello World",
			},
			renderOptions: []RenderOption{
				WithName("outlet"),
				WithTarget("layout"),
			},
			expectRenderOutput: []string{"<span>Hello World</span>"},
		},
		"Supports usage of WithTarget and WithName when rendering templates with nested outlets": {
			templateProvider: &OutletWithNested{
				Layout: Layout{},
				LevelOneEmbed: LevelOneEmbed{
					LevelTwoEmbed: LevelTwoEmbed{
						DefField: "Hello World",
					},
				},
			},
			renderOptions: []RenderOption{
				WithName("outlet"),
				WithTarget("layout"),
			},
			expectRenderOutput: []string{"<span>Hello World</span>"},
		},
		"Supports usage of WithTarget and WithName when rendering layouts with nested templates": {
			templateProvider: &OutletWithNestedLayout{
				LayoutWithNested: LayoutWithNested{
					DefinedField: DefinedField{
						DefField: "Hi",
					},
				},
				Content: "Hello World",
			},
			renderOptions: []RenderOption{
				WithName("outlet"),
				WithTarget("layout"),
			},
			expectRenderOutput: []string{"<title>Hi</title>\\n<span>Hello World</span>"},
		},
		"Supports usage of $ dot reference within range scopes": {
			templateProvider: &DollarSignWithinRange{
				DefList: []string{"1", "2"},
				DefStr:  "Hello",
			},
			expectRenderOutput: []string{"HelloHello"},
		},
		"Supports usage of $ dot reference within an if within range scopes": {
			templateProvider: &DollarSignWithinIfWithinRange{
				DefList: []string{"Hello", "World"},
				DefStr:  "Hello",
			},
			expectRenderOutput: []string{"PASS", "FAIL"},
		},

		// these are test cases for the compiler's built-in analyzers
		"Catches usage of {{ template }} statements containing undefined template names": {
			templateProvider:    &UndefinedTemplate{},
			expectCompileErrMsg: "template \"undefined\" is not provided",
		},
		"Catches usage of {{ template }} statements without a pipeline": {
			templateProvider: &NoPipeline{
				LevelOneEmbed: LevelOneEmbed{},
			},
			expectCompileErrMsg: "template \"one\" is not invoked with a pipeline",
		},
		"Catches usage of {{ if }} statements containing non-bool types": {
			templateProvider:    &AnyTypeIf{DefIf: 0},
			expectCompileErrMsg: "field \".DefIf\" is not type bool: got int",
		},
		"Catches usage of {{ if }} statements containing undefined fields": {
			templateProvider:    &UndefinedIf{},
			expectCompileErrMsg: "field \".UndIf\" not defined",
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
			err = tmpl.Render(&buf, tc.templateProvider, tc.renderOptions...)
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
