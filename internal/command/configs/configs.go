package configs

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"strings"

	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var config = struct {
	OptionsName        string
	AllExport          bool
	FuncWithOptionName bool
	GenAppend          bool
	Output             string
}{
	AllExport: true,
}

func FlagSet(set *pflag.FlagSet) {
	// 生成选项名称，默认使用函数名称生成。
	set.StringVarP(&config.OptionsName, "options-name", "n",
		config.OptionsName,
		"Generate option name, which is generated by default using function name.")

	// 导出所有字段选项设置。如果设置为 false，则不会导出小写字段。
	set.BoolVarP(&config.AllExport, "all-export", "e",
		config.AllExport,
		"Export all field option settings. If set to false, lowercase fields will not be exported.",
	)
	// 决定生成的设置函数的名字是否有选项名，用来有多个选项重复
	set.BoolVarP(&config.FuncWithOptionName, "with-option-name", "f",
		config.FuncWithOptionName,
		"Decide whether the name of the generated setting function has an option name, which is used to have multiple options for repetition",
	)
	// 决定slice参数是否生成Append方法.
	set.BoolVarP(&config.GenAppend, "gen-slice-append", "a", config.GenAppend,
		"decide whether generate append method for slice option.",
	)
	// 生成文件名
	set.StringVarP(&config.Output, "output", "o", config.Output,
		"decice output file name.",
	)
}

// Version option command version
var Version string = "0.0.2"

func RunCommand(cmd *cobra.Command, args ...string) {
	// parse file from env, which was seted by go generate tool.
	pkg, optSt := parseGoGenerate()
	// util.Dump(optSt)
	optSt.fixStruct()
	generate(pkg, optSt)
	return
}

// parseGoGenerate parse and ready generate struct.
func parseGoGenerate() (pkg *goparse.Package, optSt *optionStruct) {

	// parse file from env, which was seted by go generate tool.
	pkg, err := goparse.ParseGeneratePackage()
	util.FatalIfErr(err, "parse go generate file failed")
	node, cm, err := pkg.GetGenerateNode()
	util.FatalIfErr(err, "find generate ast node failed")

	// document and comment helper func
	foreachComment := func(node ast.Node, fc func(g *ast.CommentGroup)) {
		c, ok := cm[node]
		if !ok {
			return
		}
		for _, v := range c {
			fc(v)
		}
		return
	}

	// Only receive func declare.
	fdecl, ok := node.(*ast.FuncDecl)
	if !ok {
		util.Dump(node)
		log.Fatal("find ast node is not func type")
	}
	// Only allow func has one statement
	if len(fdecl.Body.List) != 1 {
		log.Fatal("func not only have one stmt")
	}
	// the only one clause must be return statement
	stmt, ok := fdecl.Body.List[0].(*ast.ReturnStmt)
	if !ok {
		log.Fatal("Only allow return expr in class option declaration function")
	}
	// only allow return one values
	if len(stmt.Results) != 1 {
		log.Fatal("only allow return one values")
	}
	// return value must be map literal
	result, ok := stmt.Results[0].(*ast.CompositeLit)
	if !ok {
		log.Fatal("Only allow return map literal value")
	}
	optSt = &optionStruct{}
	// from func name
	optSt.FromFunc = fdecl.Name.Name
	optSt.Name = optSt.FromFunc
	// node document
	foreachComment(node, func(g *ast.CommentGroup) {
		if len(optSt.Document) > 0 {
			optSt.Document += "\n"
		}
		optSt.Document += g.Text()
	})
	// composite elements list
	for k, elt := range result.Elts {
		// element must be key/value pair literal.
		kvexpr, ok := elt.(*ast.KeyValueExpr)
		if !ok {
			util.Dump(elt)
			log.Fatal("return value index", k, "is not key/value pair")
		}
		// key must be basic literal.(string)
		key, ok := kvexpr.Key.(*ast.BasicLit)
		if !ok {
			util.Dump(kvexpr.Key)
			log.Fatal("return value index", k, "key is not basic type(string)")
		}
		// check literal type
		if key.Kind != token.STRING {
			util.Dump(kvexpr.Key)
			log.Fatal("return value index", k, "key is not string type")
		}

		// if !token.IsExported(key.Value) {}

		// key name
		field := new(optionField)
		optSt.Fields = append(optSt.Fields, field)
		field.Name = key.Value
		// field document
		foreachComment(elt, func(g *ast.CommentGroup) {
			if len(field.Document) > 0 {
				field.Document += "\n"
			}
			field.Document += g.Text()
		})
		// maybe value comment
		field.FieldType = FieldTypeVar

		switch val := kvexpr.Value.(type) {
		case *ast.BasicLit:
			// 基础类型常量
			foreachComment(kvexpr.Value, func(g *ast.CommentGroup) {
				field.Comment = append(field.Comment, g.Text())
			})
			field.Body = val.Value
			switch val.Kind {
			case token.INT:
				field.Type = "int"
			case token.FLOAT:
				field.Type = "float"
			case token.CHAR:
				field.Type = "byte"
			case token.STRING:
				field.Type = "string"
			default:
				log.Fatal("filed ", field.Name, " not support value.",
					fmt.Sprintf("%T %#v", val.Kind, val.Kind),
				)
			}
		case *ast.CompositeLit:
			// 复合字面量
			field.Type = goparse.Format(pkg.Fset(), val.Type)
			// set body and comment
			if err := convertCompositeLitBody(pkg, cm, val, field); err != nil {
				log.Fatal("filed ", field.Name, " convert failed.",
					err,
				)
			}

		case *ast.Ident:
			// 标识符,字面量
			foreachComment(kvexpr.Value, func(g *ast.CommentGroup) {
				field.Comment = append(field.Comment, g.Text())
			})
			field.Body = val.Name
			switch val.Name {
			case "true", "false":
				field.Type = "bool"
			case "nil":
				field.Type = "interface{}"
			}
		default:
			util.Dump(kvexpr.Value)
			log.Fatal("unsupport type")
		}
	}
	return
}

// convertCompositeLitBody composite lit body convert
func convertCompositeLitBody(pkg *goparse.Package, cm ast.CommentMap, val *ast.CompositeLit,
	field *optionField) (err error) {
	var data []string

	foreachComment := func(node ast.Node, fc func(g *ast.CommentGroup)) {
		c, ok := cm[node]
		if !ok {
			return
		}
		for _, v := range c {
			fc(v)
		}
		return
	}
	for k, p := range val.Elts {
		switch elt := p.(type) {
		case *ast.BasicLit:
			data = append(data, elt.Value)
			foreachComment(elt, func(g *ast.CommentGroup) {
				field.Comment = append(field.Comment, g.Text())
			})
		case *ast.Ident:
			if elt.Name == "true" || elt.Name == "false" {
				data = append(data, elt.Name)
			} else {
				err = fmt.Errorf("[%d] not support. %s", k, elt.Name)
				return
			}
			foreachComment(elt, func(g *ast.CommentGroup) {
				field.Comment = append(field.Comment, g.Text())
			})
		case *ast.KeyValueExpr:
			blKey, okKey := elt.Key.(*ast.BasicLit)
			blVal, okVal := elt.Value.(*ast.BasicLit)
			if !okKey || !okVal {
				log.Fatalf("[%d] type %s support basic types only", k, field.Type)
			}
			data = append(data, fmt.Sprintf("%s:%s", blKey.Value, blVal.Value))
			foreachComment(elt.Value, func(g *ast.CommentGroup) {
				field.Comment = append(field.Comment, g.Text())
			})
		default:
			err = fmt.Errorf("[%d] type %s not support. %T %#v", k, field.Type, elt, elt)
			return
		}
	}
	field.Body = "nil"
	if len(data) > 0 {
		field.Body = fmt.Sprintf("%s{%s}",
			field.Type, strings.Join(data, ","))
	}
	return nil
}

type FieldType int

const (
	FieldTypeFunc FieldType = iota
	FieldTypeVar
)

type optionField struct {
	Document  string
	Comment   []string
	FieldType FieldType
	Name      string
	Type      string
	Body      string

	Export bool
}

func (field *optionField) fix() {
	field.Name = strings.Trim(field.Name, "\"")
	field.Export = true
	if !token.IsExported(field.Name) {
		field.Export = false
	}
}

func (field *optionField) GenFuncName(st *optionStruct) string {
	suffix := strings.Title(field.Name)
	if config.FuncWithOptionName {
		suffix = strings.Title(st.Name) + suffix
	}
	if !field.Export {
		return "with" + suffix
	}
	return "With" + suffix
}

func (field *optionField) AppendFuncName(st *optionStruct) string {
	suffix := strings.Title(field.Name)
	if config.FuncWithOptionName {
		suffix = strings.Title(st.Name) + suffix
	}
	if !field.Export {
		return "append" + suffix
	}
	return "Append" + suffix
}

func (field *optionField) IsSlice() bool {
	return strings.HasPrefix(field.Type, "[]") &&
		!strings.Contains(field.Type, "byte")
}

func (field *optionField) SliceType() string {
	return strings.Replace(field.Type, "[]", "", 1)
}

type optionStruct struct {
	Document   string
	Comment    []string
	Name       string
	FromFunc   string
	OptionName string
	Fields     []*optionField
}

func (opt *optionStruct) fixStruct() {
	// Option Name fix
	if config.OptionsName != "" {
		opt.Name = strings.Title(config.OptionsName)
		if strings.HasSuffix(opt.Name, "Option") {
			opt.OptionName = opt.Name
			opt.Name += "s"
		} else {
			opt.OptionName = opt.Name + "Option"
		}
	} else {
		opt.Name = opt.FromFunc
		opt.Name = strings.TrimPrefix(opt.Name, "_")
		opt.Name = strings.TrimSuffix(opt.Name, "DeclareWithDefault")
		opt.Name = strings.TrimSuffix(opt.Name, "Default")
		opt.Name = strings.Title(opt.Name)
		// Option Suffix
		if strings.HasSuffix(opt.Name, "Options") {
			opt.OptionName = strings.TrimSuffix(opt.Name, "s")
		} else if strings.HasSuffix(opt.Name, "Option") {
			opt.OptionName = opt.Name
			opt.Name += "s"
		} else {
			opt.OptionName = opt.Name + "Option"
			// opt.Name += "Options"
		}
	}
	for _, f := range opt.Fields {
		f.fix()
	}
	// all export config
	if config.AllExport {
		for _, f := range opt.Fields {
			f.Name = strings.Title(f.Name)
		}
	}
}
