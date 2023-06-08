package tmpl

import "testing"

type mockTemplateProvider struct {
	templateText string
}

var _ TemplateProvider = (*mockTemplateProvider)(nil)

func (m *mockTemplateProvider) TemplateText() string {
	return m.templateText
}

func createMockProvider(templateText string) TemplateProvider {
	return &mockTemplateProvider{templateText: templateText}
}

func Test_StaticTypeChecking(t *testing.T) {
	testCases := map[string]struct {
		provider       TemplateProvider
		expectedErrMsg string
	}{
		"Handles undefined field in IfNode": {
			provider: createMockProvider(`
				{{ if .UndefinedField }}
					No bueno
				{{ end }}
			`),
			expectedErrMsg: `failed to compile template: field "UndefinedField" not defined in type *tmpl.mockTemplateProvider`,
		},
		"Handles undefined field in RangeNode": {
			provider: createMockProvider(`
				{{ range .UndefinedField }}
					No bueno
				{{ end }}
			`),
			expectedErrMsg: `failed to compile template: field "UndefinedField" not defined in type *tmpl.mockTemplateProvider`,
		},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			_, err := Compile(testCase.provider)
			if len(testCase.expectedErrMsg) != 0 && err == nil {
				t.Errorf("expected error but got none")
			} else if len(testCase.expectedErrMsg) == 0 && err != nil {
				t.Errorf("expected no error but got %v", err)
			} else if err.Error() != testCase.expectedErrMsg {
				t.Errorf("expected error message %q but got %q", testCase.expectedErrMsg, err.Error())
			}
		})
	}
}
