package tmpl

import (
	"fmt"
	"reflect"
	"text/template/parse"
)

var builtinAnalyzers = []Analyzer{
	staticTyping,
}

// staticTyping enables static type checking on templateProvider parse trees by using
// reflection on the given struct type.
var staticTyping Analyzer = func(helper *AnalysisHelper) AnalyzerFunc {
	var visited = make(map[parse.Node]bool)

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

						visited[argTyp] = true
					}
				}
			}
			break

		case *parse.RangeNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					switch argTyp := arg.(type) {
					case *parse.FieldNode:
						field := helper.GetDefinedField(argTyp.String())
						if field == nil {
							helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", argTyp.String(), val.Interface()))
						}

						// TODO: assert that this field is a slice or array

						visited[argTyp] = true
					}
				}
			}
			break

		case *parse.TemplateNode:
			if !helper.IsDefinedTemplate(typ.Name) {
				helper.AddError(node, fmt.Sprintf("template %q is not provided by struct %T or any of its embedded structs", typ.Name, val.Interface()))
			}

			break

		// FieldNode is the last node that we want to check. Give a chance for analyzers
		// higher up in the parse tree to mark them as visited.
		case *parse.FieldNode:
			if _, ok := visited[typ]; ok {
				break
			}

			field := helper.GetDefinedField(typ.String())
			if field == nil {
				helper.AddError(node, fmt.Sprintf("field %q not defined in struct %T", typ.String(), val.Interface()))
			}

			// TODO: can we make further assertions here about the type of the field?

			break

		}
	}
}
