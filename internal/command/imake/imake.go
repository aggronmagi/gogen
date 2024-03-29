package imake

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/aggronmagi/gogen/gen"
	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var _ = util.Dump

// command config
var config = struct {
	TypeNames            []string          // 指定生成的结构体名称
	StructMatch          string            // 使用正则匹配生成的结构体名称
	IgnoreUnexportStruct bool              // 忽略未导出结构体
	IgnoreUnexportMethod bool              // 忽略未导出的成员函数
	IgnoreEmptyStruct    bool              // 忽略没有函数的结构体
	IgnoreMethods        []string          // 忽略函数集合
	TrimPackage          []string          // 忽略引入包
	ToPkg                string            // 生成的package名
	Output               string            // 生成的文件名
	IfaceSufix           string            // 生成接口名称的的后缀
	IfacePrefix          string            // 生成接口名称的的前缀
	OtherStructs         []string          // 其他包引入的结构体接口
	IFaceMap             map[string]string // 接口名称映射
	Mock                 bool              // 为结构体生成mock
	MergeUnexportIFace   bool              // 将未导出的结构体接口合并
	SortByPos            bool              // 函数以定义的顺序排序
	BuildTags            []string
	match                *regexp.Regexp
}{
	ToPkg:                ".",
	IfaceSufix:           "IFace",
	IgnoreUnexportStruct: true,
	IgnoreUnexportMethod: true,
	IFaceMap:             make(map[string]string),
	SortByPos:            true,
}

// Version generate tool version
var Version string = "0.0.7"

// Flags generate tool flags
func Flags(set *pflag.FlagSet) {
	set.StringSliceVarP(&config.TypeNames, "type", "t", config.TypeNames, "list of type names; current option is mutually exclusive with `match`")
	set.StringToStringVarP(&config.IFaceMap, "replace", "r", config.IFaceMap, "replace interface name")
	set.StringVarP(&config.StructMatch, "match", "m", "", "match struct name;current option is mutually exclusive with `type`")
	set.BoolVar(&config.IgnoreUnexportStruct, "ignore-unexport-struct", config.IgnoreUnexportStruct, "is ignore unexport struct")
	set.BoolVar(&config.IgnoreUnexportMethod, "ignore-unexport-method", config.IgnoreUnexportMethod, "is ignore unexport method")
	set.BoolVar(&config.IgnoreEmptyStruct, "ignore-empty-struct", config.IgnoreEmptyStruct, "ignore empty struct(not has funcions)")
	set.StringSliceVarP(&config.IgnoreMethods, "ignore-methods", "d", config.IgnoreMethods, "ignore methods")
	set.StringSliceVar(&config.TrimPackage, "trim-package", config.TrimPackage, "trim package")
	set.StringVarP(&config.Output, "output", "o", config.Output, "output file name; default stdout")
	set.StringVar(&config.ToPkg, "to", config.ToPkg, "generated package name")
	set.StringVarP(&config.IfaceSufix, "suffix", "s", config.IfaceSufix, "add interface name suffix")
	set.StringVarP(&config.IfacePrefix, "prefix", "p", config.IfacePrefix, "add interface name suffix")
	set.StringSliceVar(&config.BuildTags, "tags", config.BuildTags, "comma-separated list of build tags to apply")
	set.BoolVar(&config.Mock, "mock", config.Mock, "generate struct mock functions")
	set.BoolVar(&config.MergeUnexportIFace, "merge", config.MergeUnexportIFace, "merge unexport struct method to interface")
	set.BoolVar(&config.SortByPos, "sort-by-pos", config.SortByPos, "sort method by code pos")
	set.StringSliceVar(&config.OtherStructs, "merge-other", config.OtherStructs, "merge other package struct or struct name")
}

// RunCommand run generate command
func RunCommand(cmd *cobra.Command, args []string) {
	////////////////////////////////////////////////////////////
	// parse args and options

	// We accept either one directory or a list of files. Which do we have?
	if len(args) == 0 {
		log.Println("not input files,package  or directory")
		cmd.Help()
		os.Exit(2)
	}
	if len(config.StructMatch) > 0 && len(config.TypeNames) > 0 {
		log.Println("Options match and type are mutually exclusive ")
	}
	if len(config.StructMatch) > 0 {
		config.match = regexp.MustCompile(config.StructMatch)
	}

	if config.ToPkg == "." {
		config.ToPkg = goparse.EnvGoPackage
	}

	// // check mockgen
	// if config.Mock {
	// 	exec.Command("command","-v", "mockgen").Run()
	// }

	// util.Dump(config)

	////////////////////////////////////////////////////////////////////////////////
	// parse packages..
	pkg, err := goparse.ParsePackage(args, config.BuildTags...)
	util.PanicIfErr(err, "parse input failed!")
	// is same package
	fromPkg := pkg.Package().Name
	// to fix not exec from go generate
	if config.ToPkg == "" {
		config.ToPkg = fromPkg
	}
	dstPkg := fromPkg
	if dstPkg == config.ToPkg {
		dstPkg = ""
	}
	// scan struct method
	data := ParsePackages(pkg, IsGenerateStruct, IsGenerateMethod, dstPkg)

	if len(data) < 1 {
		// no struct to genrate
		log.Println("not match any struct generate.")
		cmd.Help()
		os.Exit(2)
		return
	}

	// util.Dump(data)

	g := &gen.Generator{}
	g.FormatSource = gen.OptionGoimportsFormtat
	// Print the header and package clause.
	g.Printf("// Code generated by \"gogen imake\"; DO NOT EDIT.\n")
	g.Printf("// Exec: \"gogen %s\"\n// Version: %s \n", strings.Join(os.Args[1:], " "), Version)
	g.Printf("\n")
	g.Printf("package %s", config.ToPkg)
	g.Printf("\n")
	g.Println("import (")
	trimPackageNames := make([]string, 0, 8)
	for _, v := range pkg.Package().Imports {
		trimPkg := false
		for _, tpkg := range config.TrimPackage {
			if v.PkgPath == tpkg || tpkg == v.Name {
				trimPkg = true
				trimPackageNames = append(trimPackageNames, v.Name)
				break
			}
		}
		if trimPkg {
			continue
		}
		g.Printf("%s \"%s\"\n", v.Name, v.PkgPath)
	}
	config.TrimPackage = trimPackageNames
	if len(dstPkg) > 0 {
		g.Printf("%s \"%s\"\n", fromPkg, pkg.Package().PkgPath)
	}
	g.Println(")")

	keys := sortMapKey(data)

	for _, key := range keys {
		info := data[key]
		if config.MergeUnexportIFace && !token.IsExported(info.Typ) && info.compositeOnly {
			//log.Println("ignore ", info.Typ)
			continue
		}
		// log.Printf("key %s \n", key)
		// util.Dump(info, "info")
		info.Methods = sortMethod(info.Methods)
		g.PrintDoc(info.Doc)
		g.Printf("type %s interface{\n", GetIfaceName(info.Typ))
		for _, v := range info.Composites {
			if !v.IsStruct {
				// interface composite
				g.Println(v.Typ)
				continue
			}
			stInfo, stOk := data[v.Typ]
			if config.MergeUnexportIFace && !token.IsExported(v.Typ) && stOk {
				for _, method := range stInfo.Methods {
					g.PrintDoc(method.Doc)
					g.Println(method)
				}
			} else {
				g.Printf("%s\n", GetIfaceName(v.Typ))
			}
		}
		// util.Dump(info.Composites)
		for _, method := range info.Methods {
			g.PrintDoc(method.Doc)
			g.Println(method)
		}
		g.Println("}")
		g.Println()
	}

	if len(config.Output) == 0 {
		out, err := gen.OptionGoimportsFormtat(g.Buf.Bytes())
		util.FatalIfErr(err, "generate code format")
		fmt.Println(string(out))
		return
	}
	err = g.Write(config.Output)
	util.FatalIfErr(err, "format and write result")
	////////////////////////////////////////////////////////////////////////////////
	// mock
	if !config.Mock {
		return
	}
	// check mockgen version
	var mockver string
	func() {
		out := bytes.NewBuffer(nil)
		mockcmd := exec.Command("mockgen", "-version")
		mockcmd.Stdout = out
		err = mockcmd.Run()
		util.FatalIfErr(err, "check mockgen version")
		//fmt.Println("mockgen version:", string(out.Bytes()))
		mockver = string(out.Bytes())
	}()
	// generate mockgen file
	mockfile := filepath.Join(filepath.Dir(config.Output), "mockgen.go")
	func() {
		mockcmd := exec.Command("mockgen", "-source", config.Output,
			"-destination", mockfile,
			"-package", config.ToPkg,
		)
		mockcmd.Stdout = os.Stdout
		mockcmd.Stderr = os.Stderr
		util.FatalIfErr(mockcmd.Run(), "generate mock file")
	}()
	// parse mockgen file
	mockPkg, err := goparse.ParsePackage([]string{mockfile})
	util.FatalIfErr(err, "parse mockgen generate file")
	mockData := ParsePackages(mockPkg,
		// Parse Struct
		func(name string) bool {
			// not generate mock recorder
			if strings.HasSuffix(name, "Recorder") {
				return false
			}
			// only generate mock struct
			if !strings.HasPrefix(name, "Mock") {
				return false
			}

			// ignore unexport type
			name = strings.TrimPrefix(name, "Mock")
			if !token.IsExported(name) {
				return false
			}
			return true
		},
		// Parse Method
		func(name string) bool {
			// mock add functions.
			if name == "EXPECT" {
				return false
			}
			return true
		},
		"",
	)
	// util.Dump(mockData, "mockData")
	g = &gen.Generator{}
	g.FormatSource = gen.OptionGoimportsFormtat
	// Print the header and package clause.
	g.Printf("// Code generated by \"gogen imake\"; DO NOT EDIT.\n")
	g.Printf("// Exec: \"gogen %s\"\n// Version: %s \n", strings.Join(os.Args[1:], " "), Version)
	g.Printf("// mockgen: %s\n", mockver)
	g.Printf("\n")
	g.Printf("package %s", config.ToPkg)
	g.Printf("\n")
	if len(dstPkg) > 0 {
		g.Println(`
import (
	"reflect"

	"bou.ke/monkey"
	"github.com/golang/mock/gomock"`)
		g.Printf("%s \"%s\"\n", fromPkg, pkg.Package().PkgPath)
		g.Println(")")
	}

	mockKeys := sortMapKey(mockData)
	for _, key := range mockKeys {
		info := mockData[key]
		info.Methods = sortMethod(info.Methods)
		//
		var stName string = strings.TrimPrefix(info.Typ, "Mock")
		stName = strings.TrimSuffix(stName, config.IfaceSufix)
		stFull := stName
		if len(dstPkg) > 0 {
			stFull = dstPkg + "." + stName
		}

		g.Printf(`// Stub%[1]sMock stub struct %[1]s
func Stub%[1]sMock(ctl *gomock.Controller) (mock *Mock%[1]s%[2]s,st *%[3]s) {
    mock = NewMock%[1]s%[2]s(ctl)
    st = &%[3]s{}
`,
			stName, config.IfaceSufix, stFull)
		for _, method := range info.Methods {
			// no result
			if len(method.Results) < 1 {
				g.Printf(`// stub %[2]s
	monkey.PatchInstanceMethod(reflect.TypeOf(st), "%[2]s",
		func(_ *%[1]s,%[3]s) (%[4]s) {
			mock.%[2]s(%[5]s)
            return
		},
	)
`, stFull, method.Name, method.Args(), method.Rets(), method.Args2())
				continue
			}
			g.Printf(`// stub %[2]s
	monkey.PatchInstanceMethod(reflect.TypeOf(st), "%[2]s",
		func(_ *%[1]s,%[3]s) (%[4]s) {
			return mock.%[2]s(%[5]s)
		},
	)
`, stFull, method.Name, method.Args(), method.Rets(), method.Args2())
		}
		g.Printf("\nreturn\n}\n")
	}
	util.FatalIfErr(g.Write(filepath.Join(filepath.Dir(mockfile), "stub.go")),
		"write stub file")
}

func trimPkg(typ string) string {
	for _, v := range config.TrimPackage {
		if strings.Contains(typ, v+".") {
			return strings.Replace(typ, v+".", "", 1)
		}
	}
	return typ
}

func ignoreMethod(method string) bool {
	// method = strings.ToLower(method)
	for _, v := range config.IgnoreMethods {
		if v == method {
			return true
		}
	}
	return false
}

func GetIfaceName(name string) string {
	if nm, ok := config.IFaceMap[name]; ok {
		return nm
	}

	return config.IfacePrefix + name + config.IfaceSufix
}

func sortMapKey(in map[string]*StructInfo) (out []string) {
	for k, v := range in {
		if config.IgnoreEmptyStruct && len(v.Methods) == 0 {
			continue
		}
		out = append(out, k)
	}
	sort.Strings(out)
	return
}

func sortMethod(in []*StructMethod) []*StructMethod {
	sort.Slice(in, func(i, j int) bool {
		if !config.SortByPos {
			return in[i].Name < in[j].Name
		}
		if in[i].fileName == in[j].fileName {
			return in[i].fileLine < in[j].fileLine
		}
		return in[i].fileName < in[j].fileName
	})
	dst := make([]*StructMethod, 0, len(in))
	for _, v := range in {
		if ignoreMethod(v.Name) {
			continue
		}
		dst = append(dst, v)
	}
	return dst
}

// IsGenerateStruct 是否生成结构体
func IsGenerateStruct(name string) bool {
	// 优先使用指定具体生成的参数
	if len(config.TypeNames) > 0 {
		for _, v := range config.TypeNames {
			if v == name {
				return true
			}
		}
		return false
	}

	// 忽略未导出
	if config.IgnoreUnexportStruct &&
		!token.IsExported(name) {
		return false
	}

	// 匹配
	if config.match != nil {
		return config.match.Match([]byte(name))
	}

	// 默认生成全部结构体
	return true
}

// IsGenerateMethod 是否生成结构体的方法
func IsGenerateMethod(name string) bool {
	// 忽略未导出
	if config.IgnoreUnexportMethod &&
		!token.IsExported(name) {
		return false
	}
	// 默认生成全部结构体
	return true
}

// GenerateStructInterface generate struct interface
func GenerateStructInterface(g *gen.Generator, decl *ast.GenDecl, typ *ast.TypeSpec, cm ast.CommentMap) {

}
