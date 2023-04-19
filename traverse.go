package tmpl

import "text/template/parse"

// Visitor is a function that visits nodes in a parse.Tree traversal
type Visitor = func(parse.Node)

// Traverse is a depth-first traversal utility
// for all nodes in a text/template/parse.Tree
func Traverse(cur parse.Node, visitors ...Visitor) {
	switch node := cur.(type) {
	case *parse.ActionNode:
		if node.Pipe != nil {
			Traverse(node.Pipe, visitors...)
		}
	case *parse.BoolNode:
	case *parse.BranchNode:
		if node.Pipe != nil {
			Traverse(node.Pipe, visitors...)
		}
		if node.List != nil {
			Traverse(node.List, visitors...)
		}
		if node.ElseList != nil {
			Traverse(node.ElseList, visitors...)
		}
	case *parse.BreakNode:
	case *parse.ChainNode:
	case *parse.CommandNode:
		if node.Args != nil {
			for _, arg := range node.Args {
				Traverse(arg, visitors...)
			}
		}
	case *parse.CommentNode:
	case *parse.ContinueNode:
	case *parse.DotNode:
	case *parse.FieldNode:
	case *parse.IdentifierNode:
	case *parse.IfNode:
		Traverse(&node.BranchNode, visitors...)
	case *parse.ListNode:
		if node.Nodes != nil {
			for _, child := range node.Nodes {
				Traverse(child, visitors...)
			}
		}
	case *parse.NilNode:
	case *parse.NumberNode:
	case *parse.PipeNode:
		if node.Cmds != nil {
			for _, cmd := range node.Cmds {
				Traverse(cmd, visitors...)
			}
		}
		if node.Decl != nil {
			for _, decl := range node.Decl {
				Traverse(decl, visitors...)
			}
		}
	case *parse.RangeNode:
		Traverse(&node.BranchNode, visitors...)
	case *parse.StringNode:
	case *parse.TemplateNode:
		if node.Pipe != nil {
			Traverse(node.Pipe, visitors...)
		}
	case *parse.TextNode:
	case *parse.VariableNode:
	case *parse.WithNode:
		Traverse(&node.BranchNode, visitors...)
	}

	for _, visitor := range visitors {
		visitor(cur)
	}
}
