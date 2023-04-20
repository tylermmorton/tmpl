package bind

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"
)

const (
	// BinderTypeFile loads all templates from a file on disk
	BinderTypeFile string = "file"
	// BinderTypeEmbed loads all templates from go:embed
	BinderTypeEmbed string = "embed"
)

type Config struct {
	BinderType string
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working directory: %v", err)
	}

	log.Printf("Binding templates in %s", cwd)

	t, ok := os.LookupEnv("TMPL_BIND_TYPE")
	if !ok {
		t = BinderTypeEmbed
	}

	entries, err := os.ReadDir(cwd)
	if err != nil {
		log.Fatalf("could not read current working directory: %v", err)
	}

	bindings := make([]TemplateBinding, 0)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if strings.HasSuffix(entry.Name(), "tmpl.bind.go") {
			continue
		} else if strings.HasSuffix(entry.Name(), "_test.go") {
			// TODO: we could generate a tmpl.bind_test.go file?
			continue
		} else if strings.HasSuffix(entry.Name(), ".go") {
			log.Printf("Analyzing %s", entry.Name())
			bindings = append(bindings, analyzeGoFile(filepath.Join(cwd, entry.Name()))...)
		}
	}

	if len(bindings) == 0 {
		log.Printf("No template bindings found, exiting")
		return
	}

	b := strings.Builder{}
	b.WriteString(fmt.Sprintf("package %s\n", filepath.Base(cwd)))

	switch t {
	case BinderTypeEmbed:
		// ikr... the tmpl lib isn't using a template???
		// TODO: replace this with a proper template
		b.WriteString("import (\n")
		b.WriteString("\t_ \"embed\"\n")
		b.WriteString(")\n")

		b.WriteString("var (\n")
		for _, binding := range bindings {
			b.WriteString(fmt.Sprintf("//go:embed %s\n", binding.FileName))
			b.WriteString(fmt.Sprintf("%sTmplText string\n", toCamelCase(binding.StructType)))
		}
		b.WriteString(")\n\n")

		for _, binding := range bindings {
			// TODO: add doc comments to the generated code
			b.WriteString(fmt.Sprintf("func (*%s) TemplateText() string {\n", binding.StructType))
			b.WriteString(fmt.Sprintf("\treturn %sTmplText\n", toCamelCase(binding.StructType)))
			b.WriteString("}\n\n")
		}
	case BinderTypeFile:
		b.WriteString("import (\n")
		b.WriteString("\t\"os\"\n")
		b.WriteString(")\n")

		for _, binding := range bindings {
			// TODO: add doc comments to the generated code
			b.WriteString(fmt.Sprintf("func (*%s) TemplateText() string {\n", binding.StructType))
			b.WriteString(fmt.Sprintf(`byt, err := os.ReadFile("%s")
				if err != nil {
					panic(err)
				}
				return string(byt)
			`, binding.FilePath))
			b.WriteString("}\n\n")
		}

	}

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		log.Fatalf("could not format tmpl.bind.go: %v", err)
	}

	err = os.WriteFile("tmpl.bind.go", src, 0644)
	if err != nil {
		log.Fatalf("could not write tmpl.bind.go: %v", err)
	}
}

type TemplateBinding struct {
	FileName string
	FilePath string

	StructType string
}

func analyzeGoFile(goFile string) []TemplateBinding {
	res := make([]TemplateBinding, 0)
	byt, err := os.ReadFile(goFile)
	if os.IsNotExist(err) || (byt != nil && len(byt) == 0) {
		panic(err)
	} else if err != nil {
		panic(err)
	} else {
		// Read the goFile and convert it to AST
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, "", string(byt), parser.ParseComments)
		if err != nil {
			panic(err)
		}

		log.Printf("%+v", f.Comments)
		for _, group := range f.Comments {
			for _, comment := range group.List {
				log.Printf("Comment: %+v", comment)
			}
		}

		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Specs == nil {
					continue
				}

				if decl.Doc != nil {
					for _, comment := range decl.Doc.List {
						if strings.HasPrefix(comment.Text, "//tmpl:bind") {
							if ts, ok := decl.Specs[0].(*ast.TypeSpec); ok {
								s := strings.Split(comment.Text, " ")
								if len(s) > 2 {
									panic("tmpl:bind can only have one argument")
								}

								res = append(res, TemplateBinding{
									FileName:   s[1],
									FilePath:   filepath.Join(filepath.Dir(goFile), s[1]),
									StructType: ts.Name.Name,
								})

								break
							}
						}
					}
				}
			}
		}
	}

	return res
}

// Converts snake_case to camelCase
func toCamelCase(inputUnderScoreStr string) (camelCase string) {
	flag := false
	for k, v := range inputUnderScoreStr {
		if k == 0 {
			camelCase = strings.ToUpper(string(inputUnderScoreStr[0]))
		} else {
			if flag {
				camelCase += strings.ToUpper(string(v))
				flag = false
			} else {
				if v == '-' || v == '_' {
					flag = true
				} else {
					camelCase += string(v)
				}
			}
		}
	}
	return
}
