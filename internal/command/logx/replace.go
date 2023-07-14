package logx

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/format"
	"go/printer"
	"go/token"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/aggronmagi/gogen/gen"
	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"golang.org/x/tools/go/ast/astutil"
)

// ReplaceZapLogger 重写logrus,log,fmt为zap包.
func ReplaceZapLogger(file *ast.File, fset *token.FileSet) bool {
	astutil.RewriteImport(fset, file, "github.com/sirupsen/logrus", cfg.DstPkg)
	addZapImport := false
	var funName string
	astutil.Apply(file, func(c *astutil.Cursor) bool {
		// if true {
		// 	if decl, ok := c.Node().(*ast.DeclStmt); ok {
		// 		util.Dump(decl, "decl")
		// 	}
		// 	if assign, ok := c.Node().(*ast.AssignStmt); ok {
		// 		util.Dump(assign, "assign")
		// 		if ident, ok := assign.Lhs[0].(*ast.Ident); ok {
		// 			util.Dump(ident, "assign.ident")
		// 			util.Dump(typeInfo.Uses[ident], "assign.ident.uses")
		// 			util.Dump(typeInfo.Defs[ident], "assign.ident.defs")
		// 		}
		// 	}
		// }
		fun, ok := c.Node().(*ast.FuncDecl)
		if !ok {
			return true
		}
		funName = fun.Name.Name
		if fun.Recv != nil {
			funName = strings.TrimPrefix(goparse.Format(fset, fun.Recv.List[0].Type), "*") + "." + funName
		}
		return true
	}, func(c *astutil.Cursor) bool {
		node := c.Node()
		callExpr, ok := node.(*ast.CallExpr)
		if !ok {
			assign, ok := node.(*ast.AssignStmt)
			if ok {
				_ = assign
				//util.Dump(assign, "assignstmt")
			}
			return true
		}
		selExpr, ok := callExpr.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}

		switch goparse.Format(fset, selExpr.X) {
		case "logrus", "fmt", "log":
			ff := fset.File(node.Pos())
			callFlag := fmt.Sprintf("[%s:%d %s] ",
				filepath.Join(filepath.Base(filepath.Dir(ff.Name())), filepath.Base(ff.Name())), ff.Line(node.Pos()),
				funName,
			)
			if replaceLoggerCallExpr(callFlag, selExpr, callExpr, fset) && !addZapImport {
				addZapImport = true
				file.Imports = append(file.Imports, &ast.ImportSpec{
					Path: &ast.BasicLit{
						Kind:  token.STRING,
						Value: `"go.uber.org/zap"`,
					},
				})
			}

		default:
			return true
		}

		return true
	})

	if !addZapImport {
		return false
	}

	buffer := &bytes.Buffer{}
	if err := format.Node(buffer, token.NewFileSet(), file); err != nil {

		// This value is defined in go/printer specifically for go/format and cmd/gofmt.
		// printerNormalizeNumbers = 1 << 30
		var config = printer.Config{Mode: printer.UseSpaces | printer.TabIndent | (1 << 30), Tabwidth: 4} // printerNormalizeNumbers
		var buf bytes.Buffer
		err2 := config.Fprint(&buf, fset, file)
		if err2 != nil {
			log.Println("config.Fprint failed:", err2)
		}
		fmt.Println(buf.String())
		log.Fatalln("Replace Error:", err)
	}
	//
	data, err := gen.OptionGoimportsFormtat(buffer.Bytes())
	if err != nil {
		fmt.Println(buffer.String())
	}
	util.FatalIfErr(err, "format code failed")
	if cfg.Stdout {
		fmt.Fprintln(os.Stdout, string(data))
	} else {
		ioutil.WriteFile(fset.File(file.FileStart).Name(), data, 0644)
	}
	fmt.Println("replace ", fset.File(file.FileStart).Name())
	return true
}

var _ = format.Node

func replaceLoggerCallExpr(callFlag string, selExpr *ast.SelectorExpr, callExpr *ast.CallExpr, fset *token.FileSet) bool {
	//util.Dump(callExpr, callFlag)
	// 修改成logx包
	selExpr.X = &ast.Ident{Name: cfg.pkgName}
	// 调用接口分类
	funcAct := 0   // 0: print,println等 1: printf
	funcLevel := 0 // 0: debug 1:info ....
	pkg := goparse.Format(fset, selExpr.X)
	switch selExpr.Sel.Name {
	case "Print", "Println", "Debug", "Trace", "Traceln":
		if pkg == "log" {
			funcLevel = 1
		}
	case "Printf", "Debugf", "Tracef":
		if pkg == "log" {
			funcLevel = 1
		}
		funcAct = 1
	case "Info", "Infoln":
		funcLevel = 1
	case "Infof":
		funcLevel = 1
		funcAct = 1
	case "Warn", "Warning", "Warnln", "Warningln":
		funcLevel = 2
	case "Warnf", "Warningf":
		funcLevel = 2
		funcAct = 1
	case "Error", "Errorln":
		funcLevel = 3
	case "Errorf":
		funcLevel = 3
		funcAct = 1
	case "Panic", "Panicln":
		funcLevel = 4
	case "Panicf":
		funcLevel = 4
		funcAct = 1
	case "Fatal", "Fatalln":
		funcLevel = 5
	case "Fatalf":
		funcLevel = 5
		funcAct = 1
	default:
		util.Dump(callExpr, callFlag+" Not Support")
		return false
	}

	switch funcLevel {
	case 0:
		selExpr.Sel.Name = "Debug"
	case 1:
		selExpr.Sel.Name = "Info"
	case 2:
		selExpr.Sel.Name = "Warn"
	case 3:
		selExpr.Sel.Name = "Error"
	case 4:
		selExpr.Sel.Name = "DPanic"
	case 5:
		selExpr.Sel.Name = "Fatal"
	}
	//
	lastArgs := callExpr.Args
	newArgs := make([]ast.Expr, 0, len(lastArgs)+2)
	switch funcAct {
	case 0: // print/println like
		index := 0
		if first, ok := lastArgs[0].(*ast.BasicLit); ok && first.Kind == token.STRING {
			newArgs = append(newArgs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: wrapString(callFlag + trimString(first.Value)),
			})
			index = 1
		} else {
			newArgs = append(newArgs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: wrapString(callFlag),
			})
		}
		_ = index
		for ; index < len(lastArgs); index++ {
			if arg := swtichToZapArg(lastArgs[index], fset); arg != nil {
				newArgs = append(newArgs, arg)
			}
		}
		// callExpr.Args = newArgs
	case 1: // printf like
		if first, ok := lastArgs[0].(*ast.BasicLit); ok && first.Kind == token.STRING {
			newArgs = append(newArgs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: wrapString(callFlag + trimPrintfString(first.Value)),
			})
		} else {
			newArgs = append(newArgs, &ast.BasicLit{
				Kind:  token.STRING,
				Value: wrapString(callFlag),
			})
		}
		for index := 1; index < len(lastArgs); index++ {
			if arg := swtichToZapArg(lastArgs[index], fset); arg != nil {
				newArgs = append(newArgs, arg)
			}
		}
	}

	// util.Dump(selExpr, goparse.Format(pkg.Fset(), selExpr))
	callExpr.Args = newArgs
	return true
}

func wrapString(in string) string {
	return `"` + in + `"`
}
func trimString(in string) string {
	return strings.TrimSpace(strings.Trim(in, `"`))
}

func trimPrintfString(in string) string {
	in = strings.Trim(in, `"`)
	return strings.ReplaceAll(in, "%", "$")
}

func swtichToZapArg(arg ast.Expr, fset *token.FileSet) ast.Expr {
	switch arg := arg.(type) {
	case *ast.BasicLit:
		switch arg.Kind {
		case token.STRING:
			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "zap",
					},
					Sel: &ast.Ident{
						Name: "String",
					},
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: arg.Value,
					},
					&ast.BasicLit{
						Kind:  arg.Kind,
						Value: arg.Value,
					},
				},
			}
		case token.INT:
			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "zap",
					},
					Sel: &ast.Ident{
						Name: "Int",
					},
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: wrapString(arg.Value),
					},
					&ast.BasicLit{
						Kind:  arg.Kind,
						Value: arg.Value,
					},
				},
			}
		case token.FLOAT:
			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "zap",
					},
					Sel: &ast.Ident{
						Name: "Float64",
					},
				},
				Args: []ast.Expr{
					&ast.BasicLit{
						Kind:  token.STRING,
						Value: wrapString(arg.Value),
					},
					&ast.BasicLit{
						Kind:  arg.Kind,
						Value: arg.Value,
					},
				},
			}
		default:
			return arg
		}
	case *ast.Ident:
		// util.Dump(arg, "ident")
		// if obj, ok := typeInfo.Defs[arg]; ok {
		// 	util.Dump(obj.Type(), "ident.type-1 "+arg.Name)
		// }
		// if obj, ok := typeInfo.Uses[arg]; ok {
		// 	util.Dump(obj.Type(), "ident.type-2 "+arg.Name)
		// }
		// if arg.Obj != nil {
		// 	util.Dump(arg.Obj, "ident.obj")
		// 	util.Dump(arg.Obj.Decl, "ident.obj.decl")
		// 	if assign, ok := arg.Obj.Decl.(*ast.AssignStmt); ok && len(assign.Rhs) > 0 {
		// 		//util.Dump(assign.Rhs, "ident.obj.decl")
		// 		if call, ok := assign.Rhs[0].(*ast.CallExpr); ok {
		// 			util.Dump(call, "ident.obj.decl.rhs0.call")
		// 		}
		// 	}
		// }
		if arg.Name == "err" {
			return &ast.CallExpr{
				Fun: &ast.SelectorExpr{
					X: &ast.Ident{
						Name: "zap",
					},
					Sel: &ast.Ident{
						Name: "Error",
					},
				},
				Args: []ast.Expr{
					&ast.Ident{
						Name: arg.Name,
					},
				},
			}

		}
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "zap",
				},
				Sel: &ast.Ident{
					Name: "Any",
				},
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: wrapString(arg.Name),
				},
				&ast.Ident{
					Name: arg.Name,
				},
			},
		}

	case *ast.CallExpr:
		// 	util.Dump(arg, "call")
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "zap",
				},
				Sel: &ast.Ident{
					Name: "Any",
				},
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: wrapString(goparse.Format(fset, arg.Fun)),
				},
				arg,
			},
		}
	default:
		// util.Dump(arg, "other")
		return &ast.CallExpr{
			Fun: &ast.SelectorExpr{
				X: &ast.Ident{
					Name: "zap",
				},
				Sel: &ast.Ident{
					Name: "Any",
				},
			},
			Args: []ast.Expr{
				&ast.BasicLit{
					Kind:  token.STRING,
					Value: wrapString(goparse.Format(fset, arg)),
				},
				arg,
			},
		}
	}
}
