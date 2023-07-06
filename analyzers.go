package tmpl

import (
	"context"
	"fmt"
	"reflect"
	"text/template/parse"
)

type key string

const (
	visitedMapKey key = "visited"
)

func setVisited(ctx context.Context, node parse.Node) context.Context {
	if m, ok := ctx.Value(visitedMapKey).(map[parse.Node]bool); ok {
		m[node] = true
	} else {
		return context.WithValue(ctx, visitedMapKey, map[parse.Node]bool{node: true})
	}
	return ctx
}

func isVisited(ctx context.Context, node parse.Node) bool {
	if m, ok := ctx.Value(visitedMapKey).(map[parse.Node]bool); ok {
		return m[node]
	}
	return false
}

var builtinAnalyzers = []Analyzer{
	staticTyping,
}

func staticTypingRecursive(prefix string, val reflect.Value, node parse.Node, helper *AnalysisHelper) {
	switch nodeTyp := node.(type) {
	case *parse.IfNode:
		for _, cmd := range nodeTyp.Pipe.Cmds {
			if len(cmd.Args) == 1 {
				switch argTyp := cmd.Args[0].(type) {
				case *parse.FieldNode:
					if isVisited(helper.ctx, argTyp) {
						break
					}
					typ := prefix + argTyp.String()
					field := helper.GetDefinedField(typ)
					if field == nil {
						helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", typ, val.Interface()))
					} else if kind, ok := field.IsKind(reflect.Bool); !ok {
						helper.AddError(node, fmt.Sprintf("field %q is not type bool: got %s", typ, kind))
					}
					helper.WithContext(setVisited(helper.Context(), argTyp))
				}
			} else {
				// this is a pipeline like {{ if eq .Arg "foo" }}
				if arg, ok := cmd.Args[0].(*parse.IdentifierNode); ok {
					switch arg.Ident {
					// TODO: generalize this to all function calls instead of just builtins
					case "eq", "ne", "lt", "le", "gt", "ge":
						if len(cmd.Args) != 3 {
							helper.AddError(node, fmt.Sprintf("invalid number of arguments for %q: expected 3, got %d", arg.Ident, len(cmd.Args)))
						}

						kind := make([]reflect.Kind, 2)
						for i, arg := range cmd.Args[1:] {
							switch argTyp := arg.(type) {
							case *parse.FieldNode:
								typ := prefix + argTyp.String()
								field := helper.GetDefinedField(typ)
								if field == nil {
									helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", typ, val.Interface()))
								} else {
									kind[i] = field.GetKind()
								}
								helper.WithContext(setVisited(helper.Context(), argTyp))
								break

							case *parse.StringNode:
								kind[i] = reflect.String
								break

							case *parse.NumberNode:
								if argTyp.IsInt {
									kind[i] = reflect.Int
								} else if argTyp.IsFloat {
									kind[i] = reflect.Float32 // TODO: will this break on Float64?
								} else if argTyp.IsUint {
									kind[i] = reflect.Uint
								} else if argTyp.IsComplex {
									kind[i] = reflect.Complex64
								}
							}
						}

						// check if arg1 and arg2 are comparable
						if kind[0] != kind[1] {
							helper.AddError(node, fmt.Sprintf("incompatible types for %q: %s and %s", arg.Ident, kind[0], kind[1]))
						}
					}
				}
			}
		}
		break

	case *parse.RangeNode:
		// TODO: this will break for {{ range }} statements with assignments:
		//  {{ $i, $v := range .Arg }}
		var inferTyp = prefix
		// check the type of the argument passed to range: {{ range .Arg }}
		for _, cmd := range nodeTyp.Pipe.Cmds {
			for _, arg := range cmd.Args {
				switch argTyp := arg.(type) {
				case *parse.FieldNode:
					if isVisited(helper.ctx, argTyp) {
						break
					}
					inferTyp = prefix + argTyp.String()
					field := helper.GetDefinedField(inferTyp)
					if field == nil {
						helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", argTyp.String(), val.Interface()))
					}
					helper.WithContext(setVisited(helper.Context(), argTyp))
				}
			}
		}

		// recurse on the body of the range loop using the inferred type
		Traverse(nodeTyp.List, func(node parse.Node) {
			staticTypingRecursive(inferTyp, val, node, helper)
		})

		break

	case *parse.TemplateNode:
		if !helper.IsDefinedTemplate(nodeTyp.Name) {
			helper.AddError(node, fmt.Sprintf("template %q is not provided by struct %T or any of its embedded structs", nodeTyp.Name, val.Interface()))
		} else if nodeTyp.Pipe == nil {
			helper.AddError(node, fmt.Sprintf("template %q is not invoked with a pipeline", nodeTyp.Name))
		} else if len(nodeTyp.Pipe.Cmds) == 1 {
			// TODO: here we can check the type of the pipeline
			//   if the command is a DotNode, check the type of the struct for any embedded fields
			//   if the command is a FieldNode, check the type of the field and mark it as visited
			_ = nodeTyp.Pipe.Cmds[0]
		}

		break

	// FieldNode is the last node that we want to check. Give a chance for analyzers
	// higher up in the parse tree to mark them as visited.
	case *parse.FieldNode:
		if isVisited(helper.ctx, nodeTyp) {
			break
		}

		typ := prefix + nodeTyp.String()
		field := helper.GetDefinedField(typ)
		if field == nil {
			helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", typ, val.Interface()))
		}
		helper.WithContext(setVisited(helper.Context(), nodeTyp))

		// TODO: can we make further assertions here about the type of the field?

		break
	}
}

// staticTyping enables static type checking on templateProvider parse trees by using
// reflection on the given struct type.
var staticTyping Analyzer = func(helper *AnalysisHelper) AnalyzerFunc {
	return func(val reflect.Value, node parse.Node) {
		staticTypingRecursive("", val, node, helper)
	}
}
