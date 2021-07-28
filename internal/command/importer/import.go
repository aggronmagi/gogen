// importer is a tool like stringer. it import const value and type from another package.
package importer

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"sort"
	"strings"

	"github.com/aggronmagi/gogen/gen"
	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// command config
var config = struct {
	TypeNames  []string
	ValueNames []string
	FuncsNames []string
	ToPkg      string
	Output     string
	TrimPrefix string
	BuildTags  []string
	NewFunc    bool
}{
	ToPkg: ".",
}

// Version generate tool version
var Version string = "0.0.2"

// Flags generate tool flags
func Flags(set *pflag.FlagSet) {
	set.StringSliceVarP(&config.TypeNames, "type", "t", config.TypeNames, "list of type names; must be set")
	set.StringSliceVarP(&config.ValueNames, "value", "v", config.ValueNames, "list of value names")
	set.StringSliceVarP(&config.FuncsNames, "func", "f", config.FuncsNames, "list of functions names")
	set.StringVarP(&config.Output, "output", "o", config.Output, "output file name; default <package>_import.go")
	set.StringVar(&config.ToPkg, "to", config.ToPkg, "which package be imported, extract the package from this folder")
	set.StringVarP(&config.TrimPrefix, "trimprefix", "p", config.TrimPrefix, "trim the `prefix` from the generated constant names")
	set.StringSliceVar(&config.BuildTags, "tags", config.BuildTags, "comma-separated list of build tags to apply")
	set.BoolVar(&config.NewFunc, "new", false, "import type new functions")
}

// RunCommand run generate command
func RunCommand(cmd *cobra.Command, args []string) {

	if len(config.TypeNames) < 1 && len(config.ValueNames) < 1 && len(config.FuncsNames) < 1 {
		log.Println("not set any imports flags")
		cmd.Help()
		os.Exit(2)
	}

	// We accept either one directory or a list of files. Which do we have?
	if len(args) == 0 {
		log.Println("not input files or directory")
		cmd.Help()
		os.Exit(2)
	}
	// check tags flags
	if len(args) > 1 || !util.IsDirectory(args[0]) {
		if len(config.BuildTags) != 0 {
			log.Fatal("-tags option applies only to directories, not when files are specified")
		}
	}

	pkg, err := goparse.ParsePackage(args, config.BuildTags...)
	util.PanicIfErr(err, "parse input failed!")

	g := &gen.Generator{}

	if config.ToPkg == "." {
		config.ToPkg = goparse.EnvGoPackage
	}
	fromPkg := pkg.Package().Name

	// Print the header and package clause.
	g.Printf("// Code generated by \"gogen import\"; DO NOT EDIT.\n")
	g.Printf("// Exec: \"gogen %s\"\n// Version: %s \n", strings.Join(os.Args[1:], " "), Version)
	g.Printf("\n")
	g.Printf("package %s", config.ToPkg)
	g.Printf("\n")
	g.Printf("import %s \"%s\"\n", fromPkg, pkg.Package().PkgPath)

	values := make([]Value, 0, 100)
	for _, typ := range config.TypeNames {
		// type import. normal type or const
		pkg.TypeDeclWithName(typ, func(decl *ast.GenDecl, tspec *ast.TypeSpec, cm ast.CommentMap) {
			if decl.Doc != nil {
				g.PrintDoc(decl.Doc.Text())
			}
			g.Printf("type %[1]s = %[2]s.%[1]s", typ, fromPkg)
			if tspec.Comment != nil {
				g.Println("// ", strings.TrimSpace(tspec.Comment.Text()))
			} else {
				g.Println()
			}
			return
		})
		// const value imort
		values = values[:0]
		pkg.ConstDeclValueWithType(typ,
			func(decl *ast.GenDecl, vspec *ast.ValueSpec, cm ast.CommentMap) bool {
				for _, name := range vspec.Names {
					if len(name.Name) < 1 {
						continue
					}
					// ignore
					if name.Name[0] == '_' {
						continue
					}

					// unexport value
					if !token.IsExported(name.Name) {
						continue
					}

					v := Value{
						originalName: name.Name,
					}
					if vspec.Doc != nil {
						v.doc = strings.TrimSpace(vspec.Doc.Text())
					}
					if c := vspec.Comment; c != nil {
						v.comment = strings.TrimSpace(c.Text())
					}
					v.name = strings.TrimPrefix(v.originalName, config.TrimPrefix)
					values = append(values, v)
				}
				return true
			},
		)
		// generate const value import code
		if len(values) > 0 {
			// We use stable sort so the lexically first name is chosen for equal elements.
			sort.Stable(byValue(values))

			g.Printf("\nconst (\n")
			for _, v := range values {
				g.PrintDoc(v.doc)
				g.Printf("\t%[1]s = %[2]s.%[3]s", v.name, fromPkg, v.originalName)
				if len(v.comment) > 0 {
					g.Printf(" // %s\n", v.comment)
				} else {
					g.Printf("\n")
				}
			}
			g.Printf(")\n")
		}
		// new function import
		if config.NewFunc {
			pkg.FuncDecl(func(decl *ast.FuncDecl, cm ast.CommentMap) bool {
				// ignore struct methond
				if decl.Recv != nil && len(decl.Recv.List) > 0 {
					return true
				}
				// ignore not return values
				if decl.Type.Results.NumFields() < 1 {
					return true
				}
				// first return value is dest object
				ident, ok := decl.Type.Results.List[0].Type.(*ast.Ident)
				if !ok {
					return true
				}
				// check first return value
				if ident.Name != typ {
					return true
				}

				if decl.Doc != nil {
					g.PrintDoc(decl.Doc.Text())
				}
				g.Printf("var %[1]s = %[2]s.%[1]s\n", decl.Name.String(), fromPkg)
				return true
			})
		}
	}

	// Write to file.
	outputName := config.Output
	if outputName == "" {
		outputName = fmt.Sprintf("%s_import.go", strings.ToLower(fromPkg))
	}
	err = g.Write(outputName)
	util.FatalIfErr(err, "write output file failed!")
}

// Value represents a declared constant.
type Value struct {
	originalName string // The name of the constant.
	name         string // The name with trimmed prefix.
	doc          string
	comment      string
}

func (v *Value) String() string {
	return v.originalName
}

// byValue lets us sort the constants into increasing order.
// We take care in the Less method to sort in signed or unsigned order,
// as appropriate.
type byValue []Value

func (b byValue) Len() int      { return len(b) }
func (b byValue) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byValue) Less(i, j int) bool {
	return b[i].name < b[j].name
}
