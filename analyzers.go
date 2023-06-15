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

// staticTyping enables static type checking on templateProvider parse trees by using
// reflection on the given struct type.
var staticTyping Analyzer = func(helper *AnalysisHelper) AnalyzerFunc {
	return func(val reflect.Value, node parse.Node) {
		switch typ := node.(type) {
		case *parse.IfNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					switch argTyp := arg.(type) {
					case *parse.FieldNode:
						field := helper.GetDefinedField(argTyp.String())
						if field == nil {
							helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", argTyp.String(), val.Interface()))
						} else if kind, ok := field.IsKind(reflect.Bool); !ok {
							helper.AddError(node, fmt.Sprintf("field %q is not type bool: got %s", argTyp.String(), kind))
						}
						helper.WithContext(setVisited(helper.Context(), argTyp))
					}
				}
			}
			break

		case *parse.RangeNode:
			var argPrefix string
			// check the type of the argument passed to range: {{ range Arg }}
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					switch argTyp := arg.(type) {
					case *parse.FieldNode:
						argPrefix = argTyp.String()
						field := helper.GetDefinedField(argPrefix)
						if field == nil {
							helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", argTyp.String(), val.Interface()))
						}
						helper.WithContext(setVisited(helper.Context(), argTyp))

						// TODO: assert that this field is a slice or array
					}
				}
			}

			// TODO: this is indicative of a needed refactor. this should be recursive?
			// Run a type check on the body of the range loop
			Traverse(typ.List, func(n parse.Node) {
				switch nTyp := n.(type) {
				case *parse.FieldNode:
					fqn := argPrefix + nTyp.String()
					field := helper.GetDefinedField(fqn)
					if field == nil {
						helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", fqn, val.Interface()))
					}
					helper.WithContext(setVisited(helper.Context(), nTyp))
				}
			})

			break

		case *parse.TemplateNode:
			if !helper.IsDefinedTemplate(typ.Name) {
				helper.AddError(node, fmt.Sprintf("template %q is not provided by struct %T or any of its embedded structs", typ.Name, val.Interface()))
			}

			break

		// FieldNode is the last node that we want to check. Give a chance for analyzers
		// higher up in the parse tree to mark them as visited.
		case *parse.FieldNode:
			if isVisited(helper.ctx, typ) {
				break
			}

			field := helper.GetDefinedField(typ.String())
			if field == nil {
				helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", typ.String(), val.Interface()))
			}
			helper.WithContext(setVisited(helper.Context(), typ))

			// TODO: can we make further assertions here about the type of the field?

			break

		}
	}
}
