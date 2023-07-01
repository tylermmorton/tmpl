package testdata

type TextComponent struct {
	Text string
}

func (t *TextComponent) String() string {
	return t.Text
}

func (*TextComponent) TemplateText() string {
	return "{{.}}"
}

type ScriptComponent struct {
	Source string
}

func (*ScriptComponent) TemplateText() string {
	return "<script src=\"{{.Source}}\"></script>"
}

type UndefinedTemplate struct{}

func (*UndefinedTemplate) TemplateText() string {
	return `{{ template "undefined" }}`
}

type UndefinedRange struct{}

func (*UndefinedRange) TemplateText() string {
	return `{{ range .UndList }}{{ end }}`
}

type DefinedRange struct {
	DefList []string
}

func (*DefinedRange) TemplateText() string {
	return `{{ range .DefList }}{{ . }}{{ end }}`
}

type StructRange struct {
	DefList []struct {
		DefField string
	}
}

func (*StructRange) TemplateText() string {
	return `{{ range .DefList }}{{ .DefField }}{{ end }}`
}

type EmbeddedStruct struct {
	DefField string
}

type EmbeddedField struct {
	EmbeddedStruct
}

func (*EmbeddedField) TemplateText() string {
	return `{{ .DefField }}`
}

type DefinedField struct {
	DefField string
}

func (*DefinedField) TemplateText() string {
	return `{{ .DefField }}`
}

type DefinedNestedField struct {
	Nested DefinedField
}

func (*DefinedNestedField) TemplateText() string {
	return `{{ .Nested.DefField }}`
}

type UndefinedField struct{}

func (*UndefinedField) TemplateText() string {
	return `{{ .UndField }}`
}

type UndefinedNestedField struct {
	Nested UndefinedField
}

func (*UndefinedNestedField) TemplateText() string {
	return `{{ .Nested.UndField }}`
}

type UndefinedIf struct{}

func (*UndefinedIf) TemplateText() string {
	return `{{ if .UndIf }}{{ end }}`
}

type DefinedIf struct {
	DefIf   bool
	Message string
}

func (*DefinedIf) TemplateText() string {
	return `{{ if .DefIf }}{{ .Message }}{{ end }}`
}

type AnyTypeIf struct {
	DefIf any
}

func (*AnyTypeIf) TemplateText() string {
	return `{{ if .DefIf }}{{ end }}`
}

// Tests multiple levels of embedded templates

type LevelTwoEmbed struct {
	DefField string
}

func (*LevelTwoEmbed) TemplateText() string {
	return `{{ .DefField }}`
}

type LevelOneEmbed struct {
	LevelTwoEmbed `tmpl:"two"`
}

func (*LevelOneEmbed) TemplateText() string {
	return `{{ template "two" .}}`
}

type MultiLevelEmbeds struct {
	LevelOneEmbed `tmpl:"one"`
}

func (*MultiLevelEmbeds) TemplateText() string {
	return `{{ template "one" . }}`
}

type NoPipeline struct {
	LevelOneEmbed `tmpl:"one"`
}

func (*NoPipeline) TemplateText() string {
	return `{{ template "one" }}`
}
