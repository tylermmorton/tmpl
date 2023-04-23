package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

var (
	//go:embed templates/fileprovider.tmpl.go
	fileProviderTmplText string
	//go:embed templates/textprovider.tmpl.go
	textProviderTmplText string
)

const (
	// BinderTypeFile loads all templates from a file on disk
	BinderTypeFile string = "file"
	// BinderTypeEmbed loads all templates from go:embed
	BinderTypeEmbed string = "embed"
)

type TemplateBinding struct {
	BinderType string
	FileName   string
	FilePath   string
	StructType string
	UseWatcher bool
}

func (b *TemplateBinding) TemplateText() string {
	if b.BinderType == BinderTypeEmbed {
		return textProviderTmplText
	} else if b.BinderType == BinderTypeFile {
		return fileProviderTmplText
	} else {
		panic(fmt.Sprintf("unknown binder type: %s", b.BinderType))
	}
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalf("could not get current working directory: %v", err)
	}

	log.Printf("Binding templates in %s", cwd)

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

	imports := make(map[string]string, 0)
	for _, binding := range bindings {
		switch binding.BinderType {
		case BinderTypeEmbed:
			imports["embed"] = "_"
		case BinderTypeFile:
			imports["os"] = ""
			imports["github.com/fsnotify/fsnotify"] = ""
		}
	}

	b := bytes.Buffer{}
	b.WriteString(fmt.Sprintf("package %s\n", filepath.Base(cwd)))

	b.WriteString("import (\n")
	for k, alias := range imports {
		b.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, k))
	}
	b.WriteString(")\n")

	for _, binding := range bindings {
		log.Printf("Generating binder for %s", binding.FileName)
		log.Printf("Binder type is %s", binding.BinderType)
		t, err := template.New("binder").Parse(binding.TemplateText())
		if err != nil {
			log.Fatalf("failed to parse binder template: %v", err)
		}
		t.Funcs(template.FuncMap{
			"toCamelCase": toCamelCase,
		})
		err = t.Execute(&b, &binding)
		if err != nil {
			log.Fatalf("failed to render binder template: %v", err)
		}
	}

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		log.Printf(b.String() + "\n\n")

		log.Fatalf("could not format tmpl.bind.go: %v", err)

	}

	err = os.WriteFile("tmpl.bind.go", src, 0644)
	if err != nil {
		log.Fatalf("could not write tmpl.bind.go: %v", err)
	}
}

func analyzeGoFile(goFile string) []TemplateBinding {
	t, ok := os.LookupEnv("TMPL_BIND_TYPE")
	if !ok {
		t = BinderTypeFile
	}

	res := make([]TemplateBinding, 0)
	byt, err := os.ReadFile(goFile)
	if os.IsNotExist(err) || (byt != nil && len(byt) == 0) {
		panic(err)
	} else if err != nil {
		panic(err)
	} else {
		// Read the Go File and convert it to AST
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
								b := TemplateBinding{
									FileName:   s[1],
									FilePath:   filepath.Join(filepath.Dir(goFile), s[1]),
									StructType: ts.Name.Name,
									BinderType: t,
								}

								for _, flag := range s[2:] {
									if strings.Contains(flag, "=") {
										f := strings.Split(flag, "=")[0]
										v := strings.Split(flag, "=")[1]

										switch f {
										case "watch":
											w, err := strconv.ParseBool(v)
											if err != nil {
												panic(fmt.Sprintf("failed to parse --watch value `%s` for %s", v, b.StructType))
											}
											b.UseWatcher = w
										}
									}
									if flag == "--watch" {
										b.UseWatcher = true
									}
								}

								res = append(res, b)
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
