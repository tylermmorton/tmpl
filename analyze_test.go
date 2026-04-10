package tmpl

import (
	"context"
	"reflect"
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/require"
)

// The types below are the minimal TemplateProviders needed to test the
// AnalysisHelper and Analyze APIs in isolation.

type analyzeBasicProvider struct {
	Name string
}

func (p *analyzeBasicProvider) TemplateText() string {
	return `{{ .Name }}`
}

// analyzeNestedFieldProvider tests that the helper resolves multi-segment paths.
type analyzeNestedFieldProvider struct {
	Parent analyzeNestedChild
}

type analyzeNestedChild struct {
	Value string
}

func (p *analyzeNestedFieldProvider) TemplateText() string {
	return `{{ .Parent.Value }}`
}

// analyzeProviderWithDefine contains a named template defined via {{define}}
// so tests can exercise IsDefinedTemplate with a real treeSet entry.
type analyzeProviderWithDefine struct{}

func (p *analyzeProviderWithDefine) TemplateText() string {
	return `{{ define "inner" }}hello{{ end }}{{ template "inner" . }}`
}

// TestCreateHelper covers the internal helper construction that every Analyzer
// depends on. If these invariants break, all downstream analysis will fail.
func TestCreateHelper(t *testing.T) {
	t.Run("returns a non-nil helper for a valid TemplateProvider", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)
		require.NotNil(t, helper)
	})

	t.Run("initializes context to a non-nil Background context", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)
		require.NotNil(t, helper.ctx)
	})

	t.Run("populates the treeSet with at least one parse.Tree entry", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)
		require.NotEmpty(t, helper.treeSet)
	})

	t.Run("populates the fieldTree with the provider's exported fields", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)
		require.NotNil(t, helper.fieldTree)
		node := helper.fieldTree.FindPath([]string{"Name"})
		require.NotNil(t, node, "field 'Name' should be discoverable in the field tree")
	})

	t.Run("populates the fieldTree for a nested struct field", func(t *testing.T) {
		helper, err := createHelper(&analyzeNestedFieldProvider{}, ParseOptions{})
		require.NoError(t, err)
		node := helper.fieldTree.FindPath([]string{"Parent", "Value"})
		require.NotNil(t, node, "nested path 'Parent.Value' should be in the field tree")
	})

	t.Run("errors accumulate as an empty slice, not nil", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)
		// An Analyzer calling len(helper.errors) must not panic on a fresh helper.
		require.NotNil(t, helper.errors)
		require.Empty(t, helper.errors)
	})
}

// TestAnalysisHelper tests the public methods of AnalysisHelper that Analyzer
// implementations use to communicate findings back to the analysis pipeline.
func TestAnalysisHelper(t *testing.T) {
	t.Run("Context", func(t *testing.T) {
		t.Run("default context is non-nil", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			require.NotNil(t, helper.Context())
		})

		t.Run("WithContext stores and Context returns the same context", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			type ctxKey struct{}
			ctx := context.WithValue(context.Background(), ctxKey{}, "sentinel")
			helper.WithContext(ctx)
			require.Equal(t, "sentinel", helper.Context().Value(ctxKey{}))
		})

		t.Run("WithContext called twice keeps the most recent value", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			type ctxKey struct{}
			helper.WithContext(context.WithValue(context.Background(), ctxKey{}, "first"))
			helper.WithContext(context.WithValue(context.Background(), ctxKey{}, "second"))
			require.Equal(t, "second", helper.Context().Value(ctxKey{}))
		})
	})

	t.Run("IsDefinedTemplate", func(t *testing.T) {
		t.Run("outlet is always considered defined regardless of the provider", func(t *testing.T) {
			// "outlet" is the reserved name for the layout/outlet render pattern.
			// Any analyzer logic that checks IsDefinedTemplate must treat it as defined.
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			require.True(t, helper.IsDefinedTemplate("outlet"))
		})

		t.Run("returns true for a template defined via {{define}} in the provider", func(t *testing.T) {
			helper, err := createHelper(&analyzeProviderWithDefine{}, ParseOptions{})
			require.NoError(t, err)
			require.True(t, helper.IsDefinedTemplate("inner"))
		})

		t.Run("returns false for a name that is not defined anywhere", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			require.False(t, helper.IsDefinedTemplate("nonexistent"))
		})
	})

	t.Run("GetDefinedField", func(t *testing.T) {
		helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
		require.NoError(t, err)

		t.Run("empty string returns the root FieldNode", func(t *testing.T) {
			node := helper.GetDefinedField("")
			require.NotNil(t, node)
			require.Equal(t, helper.fieldTree, node)
		})

		t.Run("dot-prefixed name resolves to the named field", func(t *testing.T) {
			node := helper.GetDefinedField(".Name")
			require.NotNil(t, node)
			require.Equal(t, "Name", node.StructField.Name)
		})

		t.Run("name without a leading dot resolves to the same node as with dot", func(t *testing.T) {
			withDot := helper.GetDefinedField(".Name")
			withoutDot := helper.GetDefinedField("Name")
			require.Equal(t, withDot, withoutDot)
		})

		t.Run("returns nil for a field path that does not exist", func(t *testing.T) {
			require.Nil(t, helper.GetDefinedField(".Nonexistent"))
		})

		t.Run("resolves a multi-segment nested path", func(t *testing.T) {
			nested, err := createHelper(&analyzeNestedFieldProvider{}, ParseOptions{})
			require.NoError(t, err)

			node := nested.GetDefinedField(".Parent.Value")
			require.NotNil(t, node)
			require.Equal(t, "Value", node.StructField.Name)
		})

		t.Run("returns nil when an intermediate segment of a nested path is missing", func(t *testing.T) {
			nested, err := createHelper(&analyzeNestedFieldProvider{}, ParseOptions{})
			require.NoError(t, err)
			require.Nil(t, nested.GetDefinedField(".Parent.Nonexistent"))
		})
	})

	t.Run("AddError", func(t *testing.T) {
		// We need a real parse.Node (one with a Position) to call AddError.
		tree := parseTree(t, `{{ .Name }}`)
		var fieldNode parse.Node
		Traverse(tree.Root, func(n parse.Node) {
			if _, ok := n.(*parse.FieldNode); ok {
				fieldNode = n
			}
		})
		require.NotNil(t, fieldNode, "expected a FieldNode in the parsed tree")

		t.Run("error is recorded and contains the supplied message", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			helper.AddError(fieldNode, "something went wrong")
			require.Len(t, helper.errors, 1)
			require.Contains(t, helper.errors[0], "something went wrong")
		})

		t.Run("error string includes the node's byte position", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			helper.AddError(fieldNode, "positional check")
			// The format is "%v: %s" where %v is node.Position().
			// We just verify something precedes the colon (i.e. the position is present).
			require.Contains(t, helper.errors[0], ":")
		})

		t.Run("multiple errors accumulate in order", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			helper.AddError(fieldNode, "first")
			helper.AddError(fieldNode, "second")
			require.Len(t, helper.errors, 2)
			require.Contains(t, helper.errors[0], "first")
			require.Contains(t, helper.errors[1], "second")
		})
	})

	t.Run("AddWarning", func(t *testing.T) {
		tree := parseTree(t, `{{ .Name }}`)
		var fieldNode parse.Node
		Traverse(tree.Root, func(n parse.Node) {
			if _, ok := n.(*parse.FieldNode); ok {
				fieldNode = n
			}
		})

		t.Run("warning is recorded and contains the supplied message", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			helper.AddWarning(fieldNode, "heads up")
			require.Len(t, helper.warnings, 1)
			require.Contains(t, helper.warnings[0], "heads up")
		})

		t.Run("warnings do not affect the error slice", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)

			helper.AddWarning(fieldNode, "just a warning")
			require.Empty(t, helper.errors)
		})
	})

	t.Run("AddFunc", func(t *testing.T) {
		t.Run("initializes a nil funcMap on the first call", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			helper.funcMap = nil

			helper.AddFunc("myFunc", func() string { return "hello" })
			require.NotNil(t, helper.funcMap)
			require.Contains(t, helper.funcMap, "myFunc")
		})

		t.Run("adds to an existing funcMap without clobbering other entries", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			helper.funcMap = nil

			helper.AddFunc("first", func() int { return 1 })
			helper.AddFunc("second", func() int { return 2 })
			require.Contains(t, helper.funcMap, "first")
			require.Contains(t, helper.funcMap, "second")
		})

		t.Run("FuncMap returns all functions added via AddFunc", func(t *testing.T) {
			helper, err := createHelper(&analyzeBasicProvider{}, ParseOptions{})
			require.NoError(t, err)
			helper.funcMap = nil

			helper.AddFunc("fn", func() bool { return true })
			fm := helper.FuncMap()
			require.Contains(t, fm, "fn")
		})
	})
}

// TestAnalyze covers the public Analyze function, which is the entry point that
// orchestrates helpers, traversal, and analyzer dispatch.
func TestAnalyze(t *testing.T) {
	t.Run("succeeds and returns a non-nil helper when no analyzers are provided", func(t *testing.T) {
		helper, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, nil)
		require.NoError(t, err)
		require.NotNil(t, helper)
	})

	t.Run("returns both a helper and an error when the analysis produces errors", func(t *testing.T) {
		// The helper is returned even on failure so callers can inspect warnings
		// and partial results (e.g. for tooling that reports all issues at once).
		alwaysErrors := Analyzer(func(helper *AnalysisHelper) AnalyzerFunc {
			return func(val reflect.Value, node parse.Node) {
				if _, ok := node.(*parse.FieldNode); ok {
					helper.AddError(node, "injected error")
				}
			}
		})

		helper, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, []Analyzer{alwaysErrors})
		require.Error(t, err)
		require.NotNil(t, helper, "helper must be non-nil even when errors are present")
	})

	t.Run("joins multiple errors into a single returned error", func(t *testing.T) {
		multiError := Analyzer(func(helper *AnalysisHelper) AnalyzerFunc {
			return func(val reflect.Value, node parse.Node) {
				if _, ok := node.(*parse.FieldNode); ok {
					helper.AddError(node, "error-alpha")
					helper.AddError(node, "error-beta")
				}
			}
		})

		_, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, []Analyzer{multiError})
		require.Error(t, err)
		require.Contains(t, err.Error(), "error-alpha")
		require.Contains(t, err.Error(), "error-beta")
	})

	t.Run("an Analyzer can register functions via AddFunc that appear in FuncMap()", func(t *testing.T) {
		// This is the mechanism by which analyzers inject helper functions into
		// compiled templates (e.g. for deferred validation at render time).
		injectFunc := Analyzer(func(helper *AnalysisHelper) AnalyzerFunc {
			helper.AddFunc("injected", func() string { return "hello" })
			return func(val reflect.Value, node parse.Node) {}
		})

		helper, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, []Analyzer{injectFunc})
		require.NoError(t, err)
		require.Contains(t, helper.FuncMap(), "injected")
	})

	t.Run("an Analyzer can thread state across node visits via the helper context", func(t *testing.T) {
		// Demonstrates the pattern for stateful analyzers: use helper.WithContext
		// and helper.Context() to accumulate data across multiple node visits.
		type countKey struct{}
		counting := Analyzer(func(helper *AnalysisHelper) AnalyzerFunc {
			return func(val reflect.Value, node parse.Node) {
				ctx := helper.Context()
				count, _ := ctx.Value(countKey{}).(int)
				helper.WithContext(context.WithValue(ctx, countKey{}, count+1))
			}
		})

		helper, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, []Analyzer{counting})
		require.NoError(t, err)
		count, ok := helper.Context().Value(countKey{}).(int)
		require.True(t, ok)
		require.Greater(t, count, 0, "the counting analyzer should have been called at least once")
	})

	t.Run("the Analyzer factory function is called once per node visit", func(t *testing.T) {
		// This is a subtle but important contract: the outer Analyzer func (the factory)
		// is invoked on every node, not just once per analysis pass. Analyzer authors
		// who do setup work in the factory must account for this.
		var factoryCallCount int
		perNodeFactory := Analyzer(func(helper *AnalysisHelper) AnalyzerFunc {
			factoryCallCount++
			return func(val reflect.Value, node parse.Node) {}
		})

		_, err := Analyze(&analyzeBasicProvider{}, ParseOptions{}, []Analyzer{perNodeFactory})
		require.NoError(t, err)
		require.Greater(t, factoryCallCount, 1,
			"the Analyzer factory is invoked once per traversed node, not once per analysis pass")
	})
}
