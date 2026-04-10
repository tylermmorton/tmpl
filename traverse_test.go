package tmpl

import (
	"fmt"
	"testing"
	"text/template/parse"

	"github.com/stretchr/testify/require"
)

// parseTree is a shared test helper that parses template text into a *parse.Tree.
// It uses SkipFuncCheck so templates containing unregistered function names (like
// those used in analyzer tests) do not fail at parse time.
func parseTree(t *testing.T, text string) *parse.Tree {
	t.Helper()
	p := parse.New("test")
	p.Mode = parse.SkipFuncCheck | parse.ParseComments
	trees := make(map[string]*parse.Tree)
	_, err := p.Parse(text, "{{", "}}", trees, nil)
	require.NoError(t, err)
	return trees["test"]
}

// collectNodeTypes traverses from root and returns the distinct set of node type
// names that were visited, expressed as their fmt %T string (e.g. "*parse.FieldNode").
func collectNodeTypes(root parse.Node) map[string]bool {
	seen := make(map[string]bool)
	Traverse(root, func(n parse.Node) {
		seen[fmt.Sprintf("%T", n)] = true
	})
	return seen
}

// TestTraverse verifies the depth-first traversal behavior of Traverse.
// Each sub-test focuses on a distinct structural property so that future
// additions to traverse.go (new node types, changed visit order) produce a
// pinpointed failure rather than a broad one.
func TestTraverse(t *testing.T) {
	t.Run("the root node is the first node visited", func(t *testing.T) {
		tree := parseTree(t, `hello`)
		var first parse.Node
		Traverse(tree.Root, func(n parse.Node) {
			if first == nil {
				first = n
			}
		})
		require.Equal(t, tree.Root, first)
	})

	t.Run("all visitors receive every node", func(t *testing.T) {
		// Multiple visitors passed to a single Traverse call must each be invoked
		// the same number of times. This is the contract that Analyzer authors rely on.
		tree := parseTree(t, `{{ .Field }}`)
		var counts [3]int
		Traverse(tree.Root,
			func(n parse.Node) { counts[0]++ },
			func(n parse.Node) { counts[1]++ },
			func(n parse.Node) { counts[2]++ },
		)
		require.Equal(t, counts[0], counts[1], "visitor 0 and visitor 1 should visit the same number of nodes")
		require.Equal(t, counts[1], counts[2], "visitor 1 and visitor 2 should visit the same number of nodes")
	})

	t.Run("a parent node is visited before its children", func(t *testing.T) {
		// Traverse is documented as depth-first. Concretely: an IfNode must appear
		// in the visit sequence before the TextNode contained in its body.
		tree := parseTree(t, `{{ if .Active }}body{{ end }}`)
		var order []string
		Traverse(tree.Root, func(n parse.Node) {
			order = append(order, fmt.Sprintf("%T", n))
		})
		ifIdx, textIdx := -1, -1
		for i, typ := range order {
			if typ == "*parse.IfNode" && ifIdx == -1 {
				ifIdx = i
			}
			if typ == "*parse.TextNode" && textIdx == -1 {
				textIdx = i
			}
		}
		require.NotEqual(t, -1, ifIdx, "expected an IfNode to be visited")
		require.NotEqual(t, -1, textIdx, "expected a TextNode to be visited")
		require.Greater(t, textIdx, ifIdx, "IfNode should be visited before the TextNode in its body")
	})

	t.Run("ActionNode and its FieldNode argument are both visited", func(t *testing.T) {
		tree := parseTree(t, `{{ .Field }}`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.ActionNode"])
		require.True(t, types["*parse.FieldNode"])
	})

	t.Run("TextNode is visited for literal content", func(t *testing.T) {
		tree := parseTree(t, `hello world`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.TextNode"])
	})

	t.Run("both branches of an if/else are visited", func(t *testing.T) {
		// Verifies that Traverse descends into the ElseList, not just the List.
		tree := parseTree(t, `{{ if .Active }}yes{{ else }}no{{ end }}`)
		var texts []string
		Traverse(tree.Root, func(n parse.Node) {
			if txt, ok := n.(*parse.TextNode); ok {
				texts = append(texts, string(txt.Text))
			}
		})
		require.Contains(t, texts, "yes", "the if-branch body should be visited")
		require.Contains(t, texts, "no", "the else-branch body should be visited")
	})

	t.Run("RangeNode and its body are visited", func(t *testing.T) {
		tree := parseTree(t, `{{ range .Items }}{{ .Name }}{{ end }}`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.RangeNode"])
		require.True(t, types["*parse.FieldNode"])
	})

	t.Run("TemplateNode and its pipe arguments are visited", func(t *testing.T) {
		tree := parseTree(t, `{{ template "foo" . }}`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.TemplateNode"])
		// The dot (.) passed as the pipeline argument should also be visited.
		require.True(t, types["*parse.DotNode"])
	})

	t.Run("CommentNode is visited when present", func(t *testing.T) {
		tree := parseTree(t, `{{/* a comment */}}`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.CommentNode"])
	})

	t.Run("all node types in a deeply nested structure are visited", func(t *testing.T) {
		// Smoke-test that nesting doesn't swallow any nodes. The template has no
		// literal text content, so ActionNode / FieldNode coverage comes from the
		// range and if conditions and from the field access in the if body.
		tree := parseTree(t, `{{ range .Items }}{{ if .Active }}{{ .Name }}{{ end }}{{ end }}`)
		types := collectNodeTypes(tree.Root)
		require.True(t, types["*parse.RangeNode"])
		require.True(t, types["*parse.IfNode"])
		require.True(t, types["*parse.FieldNode"])
		require.True(t, types["*parse.ActionNode"])
	})
}
