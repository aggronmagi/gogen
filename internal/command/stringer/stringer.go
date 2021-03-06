// Copyright 2014 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Stringer is a tool to automate the creation of methods that satisfy the fmt.Stringer
// interface. Given the name of a (signed or unsigned) integer type T that has constants
// defined, stringer will create a new self-contained Go source file implementing
//	func (t T) String() string
// The file is created in the same package and directory as the package that defines T.
// It has helpful defaults designed for use with go generate.
//
// Stringer works best with constants that are consecutive values such as created using iota,
// but creates good code regardless. In the future it might also provide custom support for
// constant sets that are bit patterns.
//
// For example, given this snippet,
//
//	package painkiller
//
//	type Pill int
//
//	const (
//		Placebo Pill = iota
//		Aspirin
//		Ibuprofen
//		Paracetamol
//		Acetaminophen = Paracetamol
//	)
//
// running this command
//
//	stringer -type=Pill
//
// in the same directory will create the file pill_string.go, in package painkiller,
// containing a definition of
//
//	func (Pill) String() string
//
// That method will translate the value of a Pill constant to the string representation
// of the respective constant name, so that the call fmt.Print(painkiller.Aspirin) will
// print the string "Aspirin".
//
// Typically this process would be run using go generate, like this:
//
//	//go:generate stringer -type=Pill
//
// If multiple constants have the same value, the lexically first matching name will
// be used (in the example, Acetaminophen will print as "Paracetamol").
//
// With no arguments, it processes the package in the current directory.
// Otherwise, the arguments must name a single directory holding a Go package
// or a set of Go source files that represent a single Go package.
//
// The -type flag accepts a comma-separated list of types so a single run can
// generate methods for multiple types. The default output file is t_string.go,
// where t is the lower-cased name of the first type listed. It can be overridden
// with the -output flag.
//
// The -linecomment flag tells stringer to generate the text of any line comment, trimmed
// of leading spaces, instead of the constant name. For instance, if the constants above had a
// Pill prefix, one could write
//
//	PillAspirin // Aspirin
//
// to suppress it in the output.
package stringer

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/constant"
	"go/types"
	"log"
	"os"
	"path/filepath"
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
	TypeNames   []string
	Output      string
	TrimPrefix  string
	LineComment bool
	BuildTags   []string
}{
	TypeNames:   []string{},
	Output:      "",
	TrimPrefix:  "",
	LineComment: false,
	BuildTags:   []string{},
}

// Version generate tool version
var Version string = "0.0.1"

// Flags generate tool flags
func Flags(set *pflag.FlagSet) {
	set.StringSliceVarP(&config.TypeNames, "type", "t", config.TypeNames, "list of type names; must be set")
	set.StringVarP(&config.Output, "output", "o", config.Output, "output file name; default srcdir/<type>_string.go")
	set.StringVarP(&config.TrimPrefix, "trimprefix", "p", config.TrimPrefix, "trim the `prefix` from the generated constant names")
	set.BoolVar(&config.LineComment, "linecomment", false, "use line comment text as printed text when present")
	set.StringSliceVar(&config.BuildTags, "tags", config.BuildTags, "comma-separated list of build tags to apply")
}

// RunCommand run generate command
func RunCommand(cmd *cobra.Command, args []string) {

	if len(config.TypeNames) < 1 {
		log.Println("not set -t or --type")
		cmd.Help()
		os.Exit(2)
	}

	// We accept either one directory or a list of files. Which do we have?
	if len(args) == 0 {
		// Default: process whole package in current directory.
		args = []string{"."}
	}

	var dir string
	// TODO(suzmue): accept other patterns for packages (directories, list of files, import paths, etc).
	if len(args) == 1 && util.IsDirectory(args[0]) {
		dir = args[0]
	} else {
		if len(config.BuildTags) != 0 {
			log.Fatal("-tags option applies only to directories, not when files are specified")
		}
		dir = filepath.Dir(args[0])
	}

	pkg, err := goparse.ParsePackage(args, config.BuildTags...)
	util.FatalIfErr(err, "parse package failed")

	g := &gen.Generator{}

	// Print the header and package clause.
	g.Printf("// Code generated by \"gogen stringer\"; DO NOT EDIT.\n")
	g.Printf("// Exec: \"gogen %s\"\n// Version: %s \n", strings.Join(os.Args[1:], " "), Version)
	g.Printf("\n")
	g.Printf("package %s", pkg.Package().Name)
	g.Printf("\n")
	g.Printf("import \"strconv\"\n") // Used by all methods.

	values := make([]Value, 0, 100)
	// Run generate for each type.
	for _, typeName := range config.TypeNames {
		// const value imort
		values = values[:0]
		pkg.ConstDeclValueWithType(typeName,
			func(decl *ast.GenDecl, vspec *ast.ValueSpec, cm ast.CommentMap) bool {
				// We now have a list of names (from one line of source code) all being
				// declared with the desired type.
				// Grab their names and actual values and store them in f.values.
				for _, name := range vspec.Names {
					if name.Name == "_" {
						continue
					}
					// This dance lets the type checker find the values for us. It's a
					// bit tricky: look up the object declared by the name, find its
					// types.Const, and extract its value.
					obj, ok := pkg.GetDefObj(name)
					if !ok {
						log.Fatalf("no value for constant %s", name)
					}
					info := obj.Type().Underlying().(*types.Basic).Info()
					if info&types.IsInteger == 0 {
						log.Fatalf("can't handle non-integer constant type %s", typeName)
					}
					value := obj.(*types.Const).Val() // Guaranteed to succeed as this is CONST.
					if value.Kind() != constant.Int {
						log.Fatalf("can't happen: constant is not an integer %s", name)
					}
					i64, isInt := constant.Int64Val(value)
					u64, isUint := constant.Uint64Val(value)
					if !isInt && !isUint {
						log.Fatalf("internal error: value of %s is not an integer: %s", name, value.String())
					}
					if !isInt {
						u64 = uint64(i64)
					}
					v := Value{
						originalName: name.Name,
						value:        u64,
						signed:       info&types.IsUnsigned == 0,
						str:          value.String(),
					}
					if c := vspec.Comment; config.LineComment && c != nil && len(c.List) == 1 {
						v.name = strings.TrimSpace(c.Text())
					} else {
						v.name = strings.TrimPrefix(v.originalName, config.TrimPrefix)
					}
					values = append(values, v)
				}
				return true
			},
		)
		//
		if len(values) == 0 {
			log.Fatalf("no values defined for type %s", typeName)
		}
		// Generate code that will fail if the constants change value.
		g.Printf("func _() {\n")
		g.Printf("\t// An \"invalid array index\" compiler error signifies that the constant values have changed.\n")
		g.Printf("\t// Re-run the stringer command to generate them again.\n")
		g.Printf("\tvar x [1]struct{}\n")
		for _, v := range values {
			g.Printf("\t_ = x[%s - %s]\n", v.originalName, v.str)
		}
		g.Printf("}\n")

		runs := splitIntoRuns(values)

		// The decision of which pattern to use depends on the number of
		// runs in the numbers. If there's only one, it's easy. For more than
		// one, there's a tradeoff between complexity and size of the data
		// and code vs. the simplicity of a map. A map takes more space,
		// but so does the code. The decision here (crossover at 10) is
		// arbitrary, but considers that for large numbers of runs the cost
		// of the linear scan in the switch might become important, and
		// rather than use yet another algorithm such as binary search,
		// we punt and use a map. In any case, the likelihood of a map
		// being necessary for any realistic example other than bitmasks
		// is very low. And bitmasks probably deserve their own analysis,
		// to be done some other day.
		switch {
		case len(runs) == 1:
			buildOneRun(g, runs, typeName)
		case len(runs) <= 10:
			buildMultipleRuns(g, runs, typeName)
		default:
			buildMap(g, runs, typeName)
		}
	}

	// Write to file.
	outputName := config.Output
	if outputName == "" {
		baseName := fmt.Sprintf("%s_string.go", config.TypeNames[0])
		outputName = filepath.Join(dir, strings.ToLower(baseName))
	}
	err = g.Write(outputName)
	util.FatalIfErr(err, "write output failed")
}

// splitIntoRuns breaks the values into runs of contiguous sequences.
// For example, given 1,2,3,5,6,7 it returns {1,2,3},{5,6,7}.
// The input slice is known to be non-empty.
func splitIntoRuns(values []Value) [][]Value {
	// We use stable sort so the lexically first name is chosen for equal elements.
	sort.Stable(byValue(values))
	// Remove duplicates. Stable sort has put the one we want to print first,
	// so use that one. The String method won't care about which named constant
	// was the argument, so the first name for the given value is the only one to keep.
	// We need to do this because identical values would cause the switch or map
	// to fail to compile.
	j := 1
	for i := 1; i < len(values); i++ {
		if values[i].value != values[i-1].value {
			values[j] = values[i]
			j++
		}
	}
	values = values[:j]
	runs := make([][]Value, 0, 10)
	for len(values) > 0 {
		// One contiguous sequence per outer loop.
		i := 1
		for i < len(values) && values[i].value == values[i-1].value+1 {
			i++
		}
		runs = append(runs, values[:i])
		values = values[i:]
	}
	return runs
}

// Value represents a declared constant.
type Value struct {
	originalName string // The name of the constant.
	name         string // The name with trimmed prefix.
	// The value is stored as a bit pattern alone. The boolean tells us
	// whether to interpret it as an int64 or a uint64; the only place
	// this matters is when sorting.
	// Much of the time the str field is all we need; it is printed
	// by Value.String.
	value  uint64 // Will be converted to int64 when needed.
	signed bool   // Whether the constant is a signed type.
	str    string // The string representation given by the "go/constant" package.
}

func (v *Value) String() string {
	return v.str
}

// byValue lets us sort the constants into increasing order.
// We take care in the Less method to sort in signed or unsigned order,
// as appropriate.
type byValue []Value

func (b byValue) Len() int      { return len(b) }
func (b byValue) Swap(i, j int) { b[i], b[j] = b[j], b[i] }
func (b byValue) Less(i, j int) bool {
	if b[i].signed {
		return int64(b[i].value) < int64(b[j].value)
	}
	return b[i].value < b[j].value
}

// Helpers

// usize returns the number of bits of the smallest unsigned integer
// type that will hold n. Used to create the smallest possible slice of
// integers to use as indexes into the concatenated strings.
func usize(n int) int {
	switch {
	case n < 1<<8:
		return 8
	case n < 1<<16:
		return 16
	default:
		// 2^32 is enough constants for anyone.
		return 32
	}
}

// declareIndexAndNameVars declares the index slices and concatenated names
// strings representing the runs of values.
func declareIndexAndNameVars(g *gen.Generator, runs [][]Value, typeName string) {
	var indexes, names []string
	for i, run := range runs {
		index, name := createIndexAndNameDecl(g, run, typeName, fmt.Sprintf("_%d", i))
		if len(run) != 1 {
			indexes = append(indexes, index)
		}
		names = append(names, name)
	}
	g.Printf("const (\n")
	for _, name := range names {
		g.Printf("\t%s\n", name)
	}
	g.Printf(")\n\n")

	if len(indexes) > 0 {
		g.Printf("var (")
		for _, index := range indexes {
			g.Printf("\t%s\n", index)
		}
		g.Printf(")\n\n")
	}
}

// declareIndexAndNameVar is the single-run version of declareIndexAndNameVars
func declareIndexAndNameVar(g *gen.Generator, run []Value, typeName string) {
	index, name := createIndexAndNameDecl(g, run, typeName, "")
	g.Printf("const %s\n", name)
	g.Printf("var %s\n", index)
}

// createIndexAndNameDecl returns the pair of declarations for the run. The caller will add "const" and "var".
func createIndexAndNameDecl(g *gen.Generator, run []Value, typeName string, suffix string) (string, string) {
	b := new(bytes.Buffer)
	indexes := make([]int, len(run))
	for i := range run {
		b.WriteString(run[i].name)
		indexes[i] = b.Len()
	}
	nameConst := fmt.Sprintf("_%s_name%s = %q", typeName, suffix, b.String())
	nameLen := b.Len()
	b.Reset()
	fmt.Fprintf(b, "_%s_index%s = [...]uint%d{0, ", typeName, suffix, usize(nameLen))
	for i, v := range indexes {
		if i > 0 {
			fmt.Fprintf(b, ", ")
		}
		fmt.Fprintf(b, "%d", v)
	}
	fmt.Fprintf(b, "}")
	return b.String(), nameConst
}

// declareNameVars declares the concatenated names string representing all the values in the runs.
func declareNameVars(g *gen.Generator, runs [][]Value, typeName string, suffix string) {
	g.Printf("const _%s_name%s = \"", typeName, suffix)
	for _, run := range runs {
		for i := range run {
			g.Printf("%s", run[i].name)
		}
	}
	g.Printf("\"\n")
}

// buildOneRun generates the variables and String method for a single run of contiguous values.
func buildOneRun(g *gen.Generator, runs [][]Value, typeName string) {
	values := runs[0]
	g.Printf("\n")
	declareIndexAndNameVar(g, values, typeName)
	// The generated code is simple enough to write as a Printf format.
	lessThanZero := ""
	if values[0].signed {
		lessThanZero = "i < 0 || "
	}
	if values[0].value == 0 { // Signed or unsigned, 0 is still 0.
		g.Printf(stringOneRun, typeName, usize(len(values)), lessThanZero)
	} else {
		g.Printf(stringOneRunWithOffset, typeName, values[0].String(), usize(len(values)), lessThanZero)
	}
}

// Arguments to format are:
//	[1]: type name
//	[2]: size of index element (8 for uint8 etc.)
//	[3]: less than zero check (for signed types)
const stringOneRun = `func (i %[1]s) String() string {
	if %[3]si >= %[1]s(len(_%[1]s_index)-1) {
		return "%[1]s(" + strconv.FormatInt(int64(i), 10) + ")"
	}
	return _%[1]s_name[_%[1]s_index[i]:_%[1]s_index[i+1]]
}
`

// Arguments to format are:
//	[1]: type name
//	[2]: lowest defined value for type, as a string
//	[3]: size of index element (8 for uint8 etc.)
//	[4]: less than zero check (for signed types)
/*
 */
const stringOneRunWithOffset = `func (i %[1]s) String() string {
	i -= %[2]s
	if %[4]si >= %[1]s(len(_%[1]s_index)-1) {
		return "%[1]s(" + strconv.FormatInt(int64(i + %[2]s), 10) + ")"
	}
	return _%[1]s_name[_%[1]s_index[i] : _%[1]s_index[i+1]]
}
`

// buildMultipleRuns generates the variables and String method for multiple runs of contiguous values.
// For this pattern, a single Printf format won't do.
func buildMultipleRuns(g *gen.Generator, runs [][]Value, typeName string) {
	g.Printf("\n")
	declareIndexAndNameVars(g, runs, typeName)
	g.Printf("func (i %s) String() string {\n", typeName)
	g.Printf("\tswitch {\n")
	for i, values := range runs {
		if len(values) == 1 {
			g.Printf("\tcase i == %s:\n", &values[0])
			g.Printf("\t\treturn _%s_name_%d\n", typeName, i)
			continue
		}
		if values[0].value == 0 && !values[0].signed {
			// For an unsigned lower bound of 0, "0 <= i" would be redundant.
			g.Printf("\tcase i <= %s:\n", &values[len(values)-1])
		} else {
			g.Printf("\tcase %s <= i && i <= %s:\n", &values[0], &values[len(values)-1])
		}
		if values[0].value != 0 {
			g.Printf("\t\ti -= %s\n", &values[0])
		}
		g.Printf("\t\treturn _%s_name_%d[_%s_index_%d[i]:_%s_index_%d[i+1]]\n",
			typeName, i, typeName, i, typeName, i)
	}
	g.Printf("\tdefault:\n")
	g.Printf("\t\treturn \"%s(\" + strconv.FormatInt(int64(i), 10) + \")\"\n", typeName)
	g.Printf("\t}\n")
	g.Printf("}\n")
}

// buildMap handles the case where the space is so sparse a map is a reasonable fallback.
// It's a rare situation but has simple code.
func buildMap(g *gen.Generator, runs [][]Value, typeName string) {
	g.Printf("\n")
	declareNameVars(g, runs, typeName, "")
	g.Printf("\nvar _%s_map = map[%s]string{\n", typeName, typeName)
	n := 0
	for _, values := range runs {
		for _, value := range values {
			g.Printf("\t%s: _%s_name[%d:%d],\n", &value, typeName, n, n+len(value.name))
			n += len(value.name)
		}
	}
	g.Printf("}\n\n")
	g.Printf(stringMap, typeName)
}

// Argument to format is the type name.
const stringMap = `func (i %[1]s) String() string {
	if str, ok := _%[1]s_map[i]; ok {
		return str
	}
	return "%[1]s(" + strconv.FormatInt(int64(i), 10) + ")"
}
`
