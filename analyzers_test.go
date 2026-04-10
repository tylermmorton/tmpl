package tmpl

import (
	"context"
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/require"
)

// ─── Template providers used by TestStaticTyping ─────────────────────────────
// Each type is deliberately minimal: one scenario per type so that the test
// that fails clearly points to the exact construct being checked.

// -- FieldNode scenarios --

type stValidField struct{ Name string }

func (*stValidField) TemplateText() string { return `{{ .Name }}` }

type stUndefinedField struct{}

func (*stUndefinedField) TemplateText() string { return `{{ .Missing }}` }

type stNestedParent struct{ Child stNestedChild }
type stNestedChild struct{ Value string }

func (*stNestedParent) TemplateText() string { return `{{ .Child.Value }}` }

type stUndefinedNestedField struct{ Child stNestedChild }

func (*stUndefinedNestedField) TemplateText() string { return `{{ .Child.Nonexistent }}` }

// -- IfNode scenarios --

type stIfBoolField struct {
	Active  bool
	Message string
}

func (*stIfBoolField) TemplateText() string { return `{{ if .Active }}{{ .Message }}{{ end }}` }

type stIfIntField struct{ Count int }

func (*stIfIntField) TemplateText() string { return `{{ if .Count }}yes{{ end }}` }

type stIfAnyField struct{ Value any }

func (*stIfAnyField) TemplateText() string { return `{{ if .Value }}yes{{ end }}` }

type stIfUndefinedField struct{}

func (*stIfUndefinedField) TemplateText() string { return `{{ if .Missing }}yes{{ end }}` }

// -- IfNode comparison operator (eq/ne/lt/le/gt/ge) scenarios --

type stIfEqStringField struct{ Name string }

func (*stIfEqStringField) TemplateText() string {
	return `{{ if eq .Name "hello" }}yes{{ end }}`
}

type stIfEqIntField struct{ Count int }

func (*stIfEqIntField) TemplateText() string {
	return `{{ if eq .Count 1 }}yes{{ end }}`
}

type stIfEqUndefinedField struct{}

func (*stIfEqUndefinedField) TemplateText() string {
	return `{{ if eq .Missing "hello" }}yes{{ end }}`
}

// -- RangeNode scenarios --

type stRangeStringSlice struct{ Items []string }

func (*stRangeStringSlice) TemplateText() string {
	return `{{ range .Items }}{{ . }}{{ end }}`
}

type stRangeUndefinedField struct{}

func (*stRangeUndefinedField) TemplateText() string {
	return `{{ range .Missing }}{{ end }}`
}

type stRangeBodyParent struct{ Items []stRangeBodyItem }
type stRangeBodyItem struct{ Name string }

func (*stRangeBodyParent) TemplateText() string {
	return `{{ range .Items }}{{ .Name }}{{ end }}`
}

type stRangeBodyUndefined struct{ Items []stRangeBodyItem }

func (*stRangeBodyUndefined) TemplateText() string {
	return `{{ range .Items }}{{ .Nonexistent }}{{ end }}`
}

// -- TemplateNode scenarios --

type stTemplateUndefined struct{}

func (*stTemplateUndefined) TemplateText() string { return `{{ template "missing" . }}` }

// stTemplateNoPipeline references a child template without passing a pipeline,
// which is required for the layout/outlet rendering pattern to work.
type stTemplateDefined struct{}

func (*stTemplateDefined) TemplateText() string { return `child content` }

type stTemplateNoPipeline struct {
	Child stTemplateDefined `tmpl:"one"`
}

func (*stTemplateNoPipeline) TemplateText() string { return `{{ template "one" }}` }

// ─── Helpers ─────────────────────────────────────────────────────────────────

// analyzeWith runs the staticTyping analyzer against tp and returns the helper
// and any joined error. It is the primary driver for all TestStaticTyping cases.
func analyzeWith(t *testing.T, tp TemplateProvider) (*AnalysisHelper, error) {
	t.Helper()
	return Analyze(tp, ParseOptions{}, []Analyzer{staticTyping})
}

// ─── TestVisitedTracking ──────────────────────────────────────────────────────

// TestVisitedTracking tests the setVisited / isVisited context helpers that
// prevent the analyzer from emitting duplicate errors for the same parse.Node.
// These helpers are used heavily inside staticTypingRecursive and are a critical
// correctness primitive for any future Analyzer that needs de-duplication.
func TestVisitedTracking(t *testing.T) {
	t.Run("a fresh Background context has no visited nodes", func(t *testing.T) {
		node := &parse.FieldNode{}
		require.False(t, isVisited(context.Background(), node))
	})

	t.Run("a node is marked visited after setVisited", func(t *testing.T) {
		node := &parse.FieldNode{}
		ctx := setVisited(context.Background(), node)
		require.True(t, isVisited(ctx, node))
	})

	t.Run("visited state is specific to the node pointer, not the node type", func(t *testing.T) {
		node1 := &parse.FieldNode{}
		node2 := &parse.FieldNode{}
		ctx := setVisited(context.Background(), node1)
		require.True(t, isVisited(ctx, node1), "node1 should be visited")
		require.False(t, isVisited(ctx, node2), "node2 should not be visited")
	})

	t.Run("setVisited on a context that already has a map mutates it in place", func(t *testing.T) {
		// Once the visited map is created by the first setVisited call, subsequent
		// calls update the same map rather than creating a new context layer. This
		// means the returned context IS the same context object and both pointers
		// see all subsequent mutations.
		node1 := &parse.FieldNode{}
		node2 := &parse.FieldNode{}
		ctx := setVisited(context.Background(), node1) // creates the map
		ctx2 := setVisited(ctx, node2)                 // mutates the existing map

		require.True(t, isVisited(ctx2, node1), "node1 should still be visible via ctx2")
		require.True(t, isVisited(ctx2, node2), "node2 should be visible via ctx2")
		// Because the map is shared, ctx also sees node2's entry.
		require.True(t, isVisited(ctx, node2), "mutation is visible through the original ctx")
	})

	t.Run("a field node processed by the IfNode handler is not re-reported by the FieldNode handler", func(t *testing.T) {
		// When {{ if .Active }} is analyzed:
		//  1. The IfNode handler checks .Active and marks it visited.
		//  2. Traverse then descends into the if-condition pipe and visits .Active again
		//     as a bare FieldNode.
		//  3. The FieldNode handler must skip it because it is already visited.
		// If de-duplication breaks, the field would generate two errors (or one false
		// "undefined" error if the bool check happened first and the FieldNode handler
		// then tries to look it up at the wrong prefix).
		helper, err := analyzeWith(t, &stIfBoolField{Active: true, Message: "hi"})
		require.NoError(t, err)
		require.Empty(t, helper.errors, "no errors should be produced for a valid bool if-condition")
	})

	t.Run("a field node inside a range body is not re-reported by the top-level FieldNode handler", func(t *testing.T) {
		// The RangeNode handler recurses on the body with the element-type prefix,
		// marks each body field as visited, and then Traverse also descends into
		// the body. Without de-duplication the body field would be checked twice —
		// once correctly with the element prefix, once incorrectly at root scope.
		helper, err := analyzeWith(t, &stRangeBodyParent{})
		require.NoError(t, err)
		require.Empty(t, helper.errors, "no errors should be produced for a valid range body field access")
	})
}

// ─── TestStaticTyping ─────────────────────────────────────────────────────────

// TestStaticTyping tests each node-type handler in staticTypingRecursive.
// Sub-tests are organized by the parse.Node type that triggers them so that
// when a new node type handler is added, its tests can be slotted in here
// without reorganizing the whole file.
func TestStaticTyping(t *testing.T) {
	t.Run("FieldNode", func(t *testing.T) {
		t.Run("valid field reference does not produce an error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stValidField{Name: "hi"})
			require.NoError(t, err)
			require.Empty(t, helper.errors)
		})

		t.Run("undefined field reference produces an error naming the missing field", func(t *testing.T) {
			helper, err := analyzeWith(t, &stUndefinedField{})
			require.Error(t, err)
			require.ErrorContains(t, err, ".Missing")
			require.Len(t, helper.errors, 1, "each undefined reference should produce exactly one error")
		})

		t.Run("valid nested field reference does not produce an error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stNestedParent{})
			require.NoError(t, err)
			require.Empty(t, helper.errors)
		})

		t.Run("undefined field in a nested path produces an error naming the full path", func(t *testing.T) {
			helper, err := analyzeWith(t, &stUndefinedNestedField{})
			require.Error(t, err)
			require.ErrorContains(t, err, ".Child.Nonexistent")
			require.Len(t, helper.errors, 1)
		})
	})

	t.Run("IfNode", func(t *testing.T) {
		t.Run("bool field used as an if-condition does not produce an error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stIfBoolField{})
			require.NoError(t, err)
			require.Empty(t, helper.errors)
		})

		t.Run("int field used as an if-condition produces a type error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stIfIntField{})
			require.Error(t, err)
			require.ErrorContains(t, err, "not type bool")
			require.ErrorContains(t, err, "int")
			require.Len(t, helper.errors, 1)
		})

		t.Run("interface field holding a non-bool value produces a type error", func(t *testing.T) {
			// The analyzer resolves the runtime Kind of an 'any' field, not just
			// its declared type, so the zero value (nil interface → invalid Kind)
			// behaves differently from a non-nil interface holding e.g. an int.
			_, err := analyzeWith(t, &stIfAnyField{Value: 42})
			require.Error(t, err)
			require.ErrorContains(t, err, "not type bool")
		})

		t.Run("undefined field used as an if-condition produces a missing-field error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stIfUndefinedField{})
			require.Error(t, err)
			require.ErrorContains(t, err, ".Missing")
			require.Len(t, helper.errors, 1)
		})

		t.Run("comparison operators", func(t *testing.T) {
			t.Run("eq with a valid string field and string literal does not produce an error", func(t *testing.T) {
				helper, err := analyzeWith(t, &stIfEqStringField{})
				require.NoError(t, err)
				require.Empty(t, helper.errors)
			})

			t.Run("eq with a valid int field and numeric literal does not produce an error", func(t *testing.T) {
				helper, err := analyzeWith(t, &stIfEqIntField{})
				require.NoError(t, err)
				require.Empty(t, helper.errors)
			})

			t.Run("eq with an undefined field produces a missing-field error", func(t *testing.T) {
				helper, err := analyzeWith(t, &stIfEqUndefinedField{})
				require.Error(t, err)
				require.ErrorContains(t, err, ".Missing")
				require.Len(t, helper.errors, 1)
			})
		})
	})

	t.Run("RangeNode", func(t *testing.T) {
		t.Run("valid range over a string slice does not produce an error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stRangeStringSlice{})
			require.NoError(t, err)
			require.Empty(t, helper.errors)
		})

		t.Run("range over an undefined field produces a missing-field error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stRangeUndefinedField{})
			require.Error(t, err)
			require.ErrorContains(t, err, ".Missing")
			require.Len(t, helper.errors, 1)
		})

		t.Run("field access in the range body resolves against the element type", func(t *testing.T) {
			// {{ range .Items }}{{ .Name }}{{ end }} — .Name must resolve as a field
			// on the element type of Items, not on the root struct.
			helper, err := analyzeWith(t, &stRangeBodyParent{})
			require.NoError(t, err)
			require.Empty(t, helper.errors)
		})

		t.Run("undefined field in the range body names the full element-relative path", func(t *testing.T) {
			helper, err := analyzeWith(t, &stRangeBodyUndefined{})
			require.Error(t, err)
			// The error path includes the range prefix so it is clear which
			// nested level is broken: ".Items.Nonexistent", not just ".Nonexistent".
			require.ErrorContains(t, err, ".Items.Nonexistent")
			require.Len(t, helper.errors, 1)
		})
	})

	t.Run("TemplateNode", func(t *testing.T) {
		t.Run("undefined template name produces an error", func(t *testing.T) {
			helper, err := analyzeWith(t, &stTemplateUndefined{})
			require.Error(t, err)
			require.ErrorContains(t, err, `"missing"`)
			require.Len(t, helper.errors, 1)
		})

		t.Run("template invoked without a pipeline produces an error", func(t *testing.T) {
			// {{ template "one" }} — missing the dot (or any pipeline) that passes
			// context into the sub-template. This is a required argument for the
			// layout/outlet render pattern.
			helper, err := analyzeWith(t, &stTemplateNoPipeline{})
			require.Error(t, err)
			require.ErrorContains(t, err, "not invoked with a pipeline")
			require.Len(t, helper.errors, 1)
		})
	})
}
