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

func FixZapLogger(file *ast.File, fset *token.FileSet) {
	modifyFlag := false
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

		if goparse.Format(fset, selExpr.X) != cfg.pkgName {
			return true
		}
		ff := fset.File(node.Pos())
		callFlag := fmt.Sprintf("[%s:%d %s] ",
			filepath.Join(filepath.Base(filepath.Dir(ff.Name())), filepath.Base(ff.Name())), ff.Line(node.Pos()),
			funName,
		)
		if fixLoggerCallExpr(callFlag, selExpr, callExpr, fset) {
			modifyFlag = true
		}

		return true
	})

	if !modifyFlag {
		return
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
	fmt.Println("fix ", fset.File(file.FileStart).Name())
}

func fixLoggerCallExpr(callFlag string, selExpr *ast.SelectorExpr, callExpr *ast.CallExpr, fset *token.FileSet) bool {
	if len(callExpr.Args) < 1 {
		callExpr.Args = append(callExpr.Args, &ast.BasicLit{
			Kind:  token.STRING,
			Value: wrapString(callFlag),
		})
		return true
	}
	//util.Dump(callExpr, callFlag)
	last, ok := callExpr.Args[0].(*ast.BasicLit)
	if !ok {
		callExpr.Args = append(callExpr.Args, &ast.BasicLit{
			Kind:  token.STRING,
			Value: wrapString(callFlag),
		})
		return true
	}
	lastMsg := trimString(last.Value)
	// 有相同的头.
	if strings.HasPrefix(lastMsg, strings.TrimSpace(callFlag)) {
		return false
	}

	//
	if strings.HasPrefix(lastMsg, "[") {
		// replace
		id := strings.IndexByte(lastMsg, ']')
		if id != -1 {
			lastMsg = lastMsg[id+1:]
			lastMsg = strings.TrimSpace(lastMsg)
		}
	}
	lastMsg = callFlag + lastMsg

	//
	arg := &ast.BasicLit{
		Kind:  token.STRING,
		Value: wrapString(lastMsg),
	}
	// util.Dump(selExpr, goparse.Format(pkg.Fset(), selExpr))
	callExpr.Args[0] = arg
	return true
}
