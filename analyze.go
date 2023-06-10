package tmpl

import (
	"errors"
	"fmt"
	"html/template"
	"reflect"
	"strings"
	"text/template/parse"
)

// AnalysisHelper is a struct that contains all the data collected
// during an analysis of a TemplateProvider.
//
// An Analysis runs in two passes. The first pass collects important
// contextual information about the template definition tree that can
// be accessed in the second pass. The second pass is the actual analysis
// of the template definition tree where errors and warnings are added.
type AnalysisHelper struct {
	//pre-analysis data
	// treeSet is a map of all templates defined in the TemplateProvider,
	// as well as all of its children.
	treeSet map[string]*parse.Tree
	// errors is a slice of Errors that occurred during analysis.
	errors []string
	// warnings is a slice of Warnings that occurred during analysis.
	warnings []string
	// funcMap is a map of functions provided by analyzers that should
	// be added before the template is executed.
	funcMap template.FuncMap

	// TODO: what if...
	// Fixers []FixerFn
}

// IsDefined returns true if the given template name is defined in the
// analysis target via {{define}}, or defined by any of its embedded templates.
func (h *AnalysisHelper) IsDefined(name string) bool {
	_, ok := h.treeSet[name]
	return ok
}

func (h *AnalysisHelper) FuncMap() template.FuncMap {
	return h.funcMap
}

func (h *AnalysisHelper) AddError(node parse.Node, err string) {
	h.errors = append(h.errors, err)
}

func (h *AnalysisHelper) AddWarning(node parse.Node, err string) {
	h.warnings = append(h.warnings, err)
}

func (h *AnalysisHelper) AddFunc(name string, fn interface{}) {
	h.funcMap[name] = fn
}

// ParseOptions controls the behavior of the tp parser used by Analyze.
type ParseOptions struct {
	LeftDelim  string
	RightDelim string
}

type AnalyzerFunc func(val reflect.Value, node parse.Node)

// Analyzer is a type that parses tp text and performs an analysis
type Analyzer func(res *AnalysisHelper) AnalyzerFunc

// Analyze uses reflection on the given TemplateProvider while also parsing the
// tp text to perform an analysis. The analysis is performed by the given
// analyzers. The analysis is returned as an AnalysisHelper struct.
func Analyze(tp TemplateProvider, opts ParseOptions, analyzers []Analyzer) (*AnalysisHelper, error) {
	helper := &AnalysisHelper{
		treeSet: make(map[string]*parse.Tree),

		errors:   make([]string, 0),
		warnings: make([]string, 0),
		funcMap:  make(template.FuncMap),
	}

	if len(opts.LeftDelim) == 0 || len(opts.RightDelim) == 0 {
		opts.LeftDelim = "{{"
		opts.RightDelim = "}}"
	}

	// collects all defined template names
	err := recurseFieldsImplementing[TemplateProvider](tp, func(tp TemplateProvider, field reflect.StructField) error {
		templateName, ok := field.Tag.Lookup("tmpl")
		if !ok {
			templateName = strings.TrimPrefix(field.Name, "*")
		}

		parser := parse.New(templateName)
		parser.Mode = parse.SkipFuncCheck | parse.ParseComments

		tmp := make(map[string]*parse.Tree)
		_, err := parser.Parse(tp.TemplateText(), opts.LeftDelim, opts.RightDelim, tmp, nil)
		if err != nil {
			return err
		}

		for k, v := range tmp {
			helper.treeSet[k] = v
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	pt := helper.treeSet[strings.TrimPrefix(fmt.Sprintf("%T", tp), "*")]
	val := reflect.ValueOf(tp)

	// Do the actual traversal and analysis of the given template provider
	Traverse(pt.Root, Visitor(func(node parse.Node) {
		for _, fn := range analyzers {
			fn(helper)(val, node)
		}
	}))

	// During runtime compilation we're only worried about errors
	if len(helper.errors) > 0 {
		errs := make([]error, 0)
		for _, err := range helper.errors {
			errs = append(errs, fmt.Errorf(err))
		}
		return helper, errors.Join(errs...)
	}

	return helper, nil
}
