package tmpl

import (
	"bytes"
	"testing"
)

type compileTextProvider struct {
	Name string
}

func (ct *compileTextProvider) TemplateText() string {
	return "Hello, {{.Name}}!"
}

type compileFileProvider struct {
	Name string
}

func (cf *compileFileProvider) TemplateFile() string {
	return "./testdata/happy.tmpl.html"
}

func Test_Compile(t *testing.T) {
	testTable := map[string]struct {
		p          TemplateProvider
		expected   string
		wantErr    bool
		wantErrMsg string
	}{
		"Successfully compiles a TemplateTextProvider": {
			p: &compileTextProvider{
				Name: "World",
			},
			expected: "Hello, World!",
		},
		"Successfully compiles a TemplateFileProvider": {
			p: &compileFileProvider{
				Name: "World",
			},
			expected: "Hello, World!",
		},
	}

	for name, test := range testTable {
		t.Run(name, func(t *testing.T) {
			buf := bytes.Buffer{}
			err := Compile(test.p).Render(&buf, &compileTextProvider{Name: "World"})
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
