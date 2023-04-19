package tmpl

import (
	"errors"
	"fmt"
	"html/template"
	"text/template/parse"
)

// ParserHook is a factory function that creates a Visitor
// scoped to a Template instance.
type ParserHook func(t *template.Template) Visitor

type Parser interface {
	// Use appends the given ParserHook to the parser.
	Use(ParserHook)
	// Parse takes a Template name and the Template text and parses it.
	//
	// Additionally, a slice of Visitors can be passed and are executed against
	// the resulting parse.Tree.
	Parse(t *template.Template, name, templateText string, visitors ...Visitor) (*template.Template, error)
}

type parser struct {
	hooks []ParserHook
}

// NewParser creates a new Parser with the given hooks
func NewParser(hooks ...ParserHook) Parser {
	return &parser{hooks}
}

func (p *parser) Use(hook ParserHook) {
	if hook != nil {
		p.hooks = append(p.hooks, hook)
	}
}

func (p *parser) Parse(t *template.Template, templateName, templateText string, v ...Visitor) (*template.Template, error) {
	parser := parse.New(templateName)
	parser.Mode = parse.SkipFuncCheck | parse.ParseComments
	treeSet := make(map[string]*parse.Tree)
	pt, err := parser.Parse(templateText, "{{", "}}", treeSet, nil)
	if err != nil {
		return nil, err
	}

	if t == nil {
		t = template.New(templateName)
	}

	for name, tree := range treeSet {
		t = template.Must(t.AddParseTree(name, tree))
	}

	// defer a panic boundary to catch errors thrown by any of the
	// given visitor functions
	defer func() {
		if r := recover(); r != nil {
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = fmt.Errorf("recovered panic thrown by visitor during parse.Tree traversal: %v", t)
			}
		}
	}()

	// append any visitors registered in the parser hooks.
	// copy the slice here so we aren't modifying v
	visitors := append(make([]Visitor, 0), v...)
	for _, fn := range p.hooks {
		visitors = append(visitors, fn(t))
	}

	Traverse(pt.Root, visitors...)

	return t, nil
}
