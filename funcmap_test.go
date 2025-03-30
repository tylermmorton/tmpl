package tmpl

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

type AddFunctionComponent struct {
	A, B int
}

func (*AddFunctionComponent) TemplateText() string {
	return `{{ add .A .B }}`
}

func (*AddFunctionComponent) TemplateFuncMap() FuncMap {
	return FuncMap{
		"add": func(a, b int) string {
			return fmt.Sprintf("%d", a+b)
		},
	}
}

type SubFunctionComponent struct {
	A, B int
}

func (*SubFunctionComponent) TemplateText() string {
	return `{{ sub .A .B }}`
}

func (*SubFunctionComponent) TemplateFuncMap() FuncMap {
	return FuncMap{
		"sub": func(a, b int) string {
			return fmt.Sprintf("%d", a-b)
		},
	}
}

type MergedFunctionComponent struct {
	A, B int
	AddFunctionComponent
	SubFunctionComponent
}

func (*MergedFunctionComponent) TemplateText() string {
	return `{{ add .A .B }}, {{ sub .A .B }}`
}

type NestedFunctionComponent struct {
	A, B   int
	Nested struct {
		AddFunctionComponent
		Nested struct {
			SubFunctionComponent
		}
	}
}

func (*NestedFunctionComponent) TemplateText() string {
	return `{{ add .A .B }}, {{ sub .A .B }}`
}

func TestCompile_FuncMapProvider(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		templateProvider := &AddFunctionComponent{
			A: 1,
			B: 2,
		}

		tmpl, err := Compile(templateProvider)
		require.NoError(t, err)

		buf := bytes.Buffer{}
		err = tmpl.Render(&buf, templateProvider)
		require.NoError(t, err)

		require.Equal(t, "3", buf.String())
	})

	t.Run("merged_func_map_providers", func(t *testing.T) {
		templateProvider := &MergedFunctionComponent{
			A: 1,
			B: 2,
		}

		tmpl, err := Compile(templateProvider)
		require.NoError(t, err)

		buf := bytes.Buffer{}
		err = tmpl.Render(&buf, templateProvider)
		require.NoError(t, err)

		require.Equal(t, "3, -1", buf.String())
	})

	t.Run("nested_func_map_providers", func(t *testing.T) {
		templateProvider := &NestedFunctionComponent{
			A: 1,
			B: 2,
		}

		tmpl, err := Compile(templateProvider)
		require.NoError(t, err)

		buf := bytes.Buffer{}
		err = tmpl.Render(&buf, templateProvider)
		require.NoError(t, err)

		require.Equal(t, "3, -1", buf.String())
	})
}
