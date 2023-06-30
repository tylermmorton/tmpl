package cmd

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

	"github.com/spf13/cobra"
)

var (
	Outfile *string
	Mode    *string
	Watch   *bool

	//go:embed templates/fileprovider.tmpl
	fileProviderTmplText string
	//go:embed templates/textprovider.tmpl
	textProviderTmplText string
)

const (
	BindPrefix string = "//tmpl:bind"

	// BinderTypeFile loads all templates from a file on disk
	BinderTypeFile string = "file"
	// BinderTypeEmbed loads all templates from go:embed
	BinderTypeEmbed string = "embed"
)

type TemplateBinding struct {
	Args       []string
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

// bindCmd represents the bind command
var bindCmd = &cobra.Command{
	Use:   "bind",
	Short: "Analyzes Go source code in search of //tmpl:bind comments and generates binder files",

	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		outfile := cmd.Flags().Lookup("outfile")
		if outfile == nil {
			return fmt.Errorf("--outfile not set and no default was provided")
		}

		fileOrPath := args[0]
		if len(fileOrPath) == 0 {
			return fmt.Errorf("no file or path argument was provided")
		}

		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current working directory: %v", err)
		}

		if strings.HasSuffix(fileOrPath, "...") {
			return bindGoPackage(filepath.Join(cwd, strings.TrimSuffix(fileOrPath, "...")), outfile.Value.String(), true)
		}

		fileOrPath = filepath.Join(cwd, fileOrPath)

		s, err := os.Stat(fileOrPath)
		if err != nil {
			return fmt.Errorf("failed to read file or path '%s': %+v", fileOrPath, err)
		}

		if s.IsDir() {
			return bindGoPackage(fileOrPath, filepath.Join(fileOrPath, outfile.Value.String()), false)
		} else {
			return bindGoFile(fileOrPath, filepath.Join(filepath.Dir(fileOrPath), outfile.Value.String()))
		}
	},
}

func init() {
	rootCmd.AddCommand(bindCmd)

	Outfile = bindCmd.Flags().String("outfile", "tmpl.gen.go", "set the output go file for template bindings")
	Watch = bindCmd.Flags().Bool("watch", false, "enable generation of `TemplateWatcher` implementations")
	Mode = bindCmd.Flags().String("mode", BinderTypeFile, "set the binder mode (embed|file)")
	if mode, ok := os.LookupEnv("TMPL_BIND_MODE"); Mode == nil && ok {
		Mode = &mode
	}
}

func analyzeGoFile(goFile string) []TemplateBinding {
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
			log.Printf("Unable to parse .go file '%s' as Go source:\n\t%+v", goFile, err)
			return res
		}

		for _, decl := range f.Decls {
			switch decl := decl.(type) {
			case *ast.GenDecl:
				if decl.Specs == nil {
					continue
				}

				if decl.Doc != nil {
					for _, comment := range decl.Doc.List {
						if strings.HasPrefix(comment.Text, BindPrefix) {
							if ts, ok := decl.Specs[0].(*ast.TypeSpec); ok {
								// TODO: refactor to separate function
								s := strings.Split(comment.Text, " ")
								b := TemplateBinding{
									Args:       s[2:],
									FileName:   s[1],
									FilePath:   filepath.Join(filepath.Dir(goFile), s[1]),
									StructType: ts.Name.Name,
									BinderType: *Mode,
									UseWatcher: *Watch,
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

func writeBinderFile(outfile string, packageName string, bindings []TemplateBinding) error {
	imports := make(map[string]string, 0)
	for _, binding := range bindings {
		switch binding.BinderType {
		case BinderTypeEmbed:
			imports["embed"] = "_"
		case BinderTypeFile:
			imports["os"] = ""
			if binding.UseWatcher {
				imports["fmt"] = ""
				imports["github.com/fsnotify/fsnotify"] = ""
			}
		}
	}

	log.Printf("Generating '%s'", outfile)

	b := bytes.Buffer{}
	b.WriteString(fmt.Sprintf("package %s\n", packageName))

	b.WriteString("import (\n")
	for k, alias := range imports {
		b.WriteString(fmt.Sprintf("\t%s \"%s\"\n", alias, k))
	}
	b.WriteString(")\n")

	for _, binding := range bindings {
		log.Printf("- write binder for %s %s", binding.FileName, strings.Join(binding.Args, " "))

		t := template.New("binder").Funcs(template.FuncMap{
			"toCamelCase": toCamelCase,
		})

		t, err := t.Parse(binding.TemplateText())
		if err != nil {
			return fmt.Errorf("could not parse binder template: %v", err)
		}

		err = t.Execute(&b, &binding)
		if err != nil {
			return fmt.Errorf("could not execute binder template: %v", err)
		}
	}

	src, err := format.Source([]byte(b.String()))
	if err != nil {
		fmt.Printf(b.String() + "\n\n")
		return fmt.Errorf("could not format binder file: %v", err)
	}

	err = os.WriteFile(outfile, src, 0644)
	if err != nil {
		return fmt.Errorf("could not write binder file: %v", err)
	}

	return nil
}

func bindGoFile(goFile string, outFile string) error {
	return writeBinderFile(outFile, filepath.Base(filepath.Dir(goFile)), analyzeGoFile(goFile))
}

func bindGoPackage(dir, outFile string, recursive bool) error {
	bindings := make([]TemplateBinding, 0)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("could not read current working directory: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() && recursive {
			err := bindGoPackage(filepath.Join(dir, entry.Name()), outFile, recursive)
			if err != nil {
				return err
			}
		} else if strings.HasSuffix(entry.Name(), ".go") {
			bindings = append(bindings, analyzeGoFile(filepath.Join(dir, entry.Name()))...)
		}
	}

	if len(bindings) == 0 {
		return nil
	}

	return writeBinderFile(filepath.Join(dir, outFile), filepath.Base(dir), bindings)
}
