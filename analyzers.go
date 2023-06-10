package tmpl

import (
	"fmt"
	"reflect"
	"strings"
	"text/template/parse"
)

func hasFieldOrMethod(val reflect.Value, name string) bool {
	switch val.Kind() {
	case reflect.Ptr:
		val = val.Elem()
		fallthrough
	case reflect.Struct:
		return val.FieldByName(name) != reflect.Value{} || val.MethodByName(name) != reflect.Value{}
	default:
		fmt.Printf("unknown kind: %s\n", val.Kind())
	}
	return false
}

// staticTyping enables static type checking on tp parse trees by using
// reflection on the given struct type.
var staticTyping Analyzer = func(helper *AnalysisHelper) AnalyzerFunc {
	return func(val reflect.Value, node parse.Node) {
		switch typ := node.(type) {
		case *parse.IfNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					ident := strings.TrimPrefix(arg.String(), ".")
					if !hasFieldOrMethod(val, ident) {
						helper.AddError(node, fmt.Sprintf("field %q not defined in type %T", ident, val.Interface()))
					}
				}
			}
		case *parse.RangeNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					ident := strings.TrimPrefix(arg.String(), ".")
					if !hasFieldOrMethod(val, ident) {
						helper.AddError(node, fmt.Sprintf("field %q not defined in type %T", ident, val.Interface()))
					}
				}
			}
		case *parse.TemplateNode:
			if !helper.IsDefined(typ.Name) {
				helper.AddError(node, fmt.Sprintf("template %q is not provided by type %T or any of its embedded templates", typ.Name, val.Interface()))
			}
		}
	}
}
