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

func hasTemplateProvider(val reflect.Value, name string) bool {
	return false
}

// staticTyping enables static type checking on template parse trees by using
// reflection on the given struct type.
var staticTyping Analyzer = func(report *AnalysisReporter) AnalyzerFunc {
	return func(val reflect.Value, node parse.Node) {
		switch typ := node.(type) {
		case *parse.IfNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					ident := strings.TrimPrefix(arg.String(), ".")
					if !hasFieldOrMethod(val, ident) {
						report.AddError(node, fmt.Sprintf("field %q not defined in type %T", ident, val.Interface()))
					}
				}
			}
		case *parse.RangeNode:
			for _, cmd := range typ.Pipe.Cmds {
				for _, arg := range cmd.Args {
					ident := strings.TrimPrefix(arg.String(), ".")
					if !hasFieldOrMethod(val, ident) {
						report.AddError(node, fmt.Sprintf("field %q not defined in type %T", ident, val.Interface()))
					}
				}
			}
		case *parse.TemplateNode:
			// TODO
			break
			//if !hasTemplateProvider(val, typ.Name) {
			//	report.AddError(node, fmt.Sprintf("template %q not defined in %T or any nested providers", typ.Name, val.Interface()))
			//}
		}
	}
}
