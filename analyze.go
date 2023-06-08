package tmpl

import (
	"errors"
	"fmt"
	"reflect"
	"text/template/parse"
)

type AnalysisReporter struct {
	// Errors is a slice of Errors that occurred during analysis.
	Errors []string `json:"Errors"`
	// Warnings is a slice of Warnings that occurred during analysis.
	Warnings []string `json:"Warnings"`

	// TODO: what if...
	// Fixers []FixerFn
}

func (res *AnalysisReporter) AddError(node parse.Node, err string) {
	res.Errors = append(res.Errors, err)
}

func (res *AnalysisReporter) AddWarning(node parse.Node, err string) {
	res.Warnings = append(res.Warnings, err)
}

func (res *AnalysisReporter) Error() string {
	errs := make([]error, 0)
	for _, err := range res.Errors {
		errs = append(errs, fmt.Errorf(err))
	}
	return errors.Join(errs...).Error()
}

type AnalysisOptions struct {
	LeftDelim  string
	RightDelim string
}

type AnalyzerFunc func(val reflect.Value, node parse.Node)

// Analyzer is a type that parses template text and performs an analysis
type Analyzer func(res *AnalysisReporter) AnalyzerFunc

// Analyze uses reflection on the given TemplateProvider while also parsing the
// template text to perform an analysis. The analysis is performed by the given
// analyzers. The analysis is returned as an AnalysisReporter struct.
func Analyze(tp TemplateProvider, opts AnalysisOptions, analyzers []Analyzer) (*AnalysisReporter, error) {
	res := &AnalysisReporter{
		Errors:   make([]string, 0),
		Warnings: make([]string, 0),
	}
	val := reflect.ValueOf(tp)
	templateName := nameFromProvider(tp)
	templateText := tp.TemplateText()

	parser := parse.New(templateName)
	parser.Mode = parse.SkipFuncCheck | parse.ParseComments
	treeSet := make(map[string]*parse.Tree)

	if len(opts.LeftDelim) == 0 || len(opts.RightDelim) == 0 {
		opts.LeftDelim = "{{"
		opts.RightDelim = "}}"
	}

	pt, err := parser.Parse(templateText, opts.LeftDelim, opts.RightDelim, treeSet, nil)
	if err != nil {
		return nil, err
	}

	Traverse(pt.Root, Visitor(func(node parse.Node) {
		for _, fn := range analyzers {
			fn(res)(val, node)
		}
	}))

	return res, nil
}
