package imake

import (
	"fmt"
	"go/ast"
	"go/token"
	"log"
	"strings"

	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
)

var _ = util.Dump

type StructField struct {
	Names []string
	Type  string
}

func (f *StructField) String() string {
	v := strings.Join(f.Names, ",")
	v += " " + f.Type
	return v
}

type StructMethod struct {
	Doc      string
	Comment  string
	Name     string
	Params   []*StructField
	Results  []*StructField
	fileName string
	fileLine int
}

func (m *StructMethod) String() string {
	params := make([]string, 0, len(m.Params))
	for _, v := range m.Params {
		params = append(params, v.String())
	}
	result := make([]string, 0, len(m.Results))
	for _, v := range m.Results {
		result = append(result, v.String())
	}
	return fmt.Sprintf("%s (%s) (%s)", m.Name,
		strings.Join(params, ","), strings.Join(result, ","))
}

func (m *StructMethod) Args() string {
	params := make([]string, 0, len(m.Params))
	for _, v := range m.Params {
		params = append(params, v.String())
	}
	return strings.Join(params, ",")
}

func (m *StructMethod) Args2() string {
	params := make([]string, 0, len(m.Params))
	for _, v := range m.Params {
		params = append(params, strings.Join(v.Names, ","))
	}
	return strings.Join(params, ",")
}

func (m *StructMethod) Rets() string {
	result := make([]string, 0, len(m.Results))
	for _, v := range m.Results {
		result = append(result, v.String())
	}
	return strings.Join(result, ",")
}

type CompositeStructInfo struct {
	Typ      string
	IsStruct bool
}

type StructInfo struct {
	Typ           string
	Composites    []*CompositeStructInfo
	Doc           string
	Comment       string
	Methods       []*StructMethod
	compositeOnly bool
}

// parse dst package. collecte structs infos
func ParsePackages(pkg *goparse.Package,
	IsGenerateStruct func(name string) bool,
	IsGenerateMethod func(name string) bool,
	dstPackage string,
) (data map[string]*StructInfo) {
	// parse dst package. collecte structs infos
	data = make(map[string]*StructInfo, 16)

	stCheck := make(map[string]struct{}, 128)

	// range type declare
	pkg.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
		if decl.Tok != token.TYPE {
			return true
		}

		for _, spec := range decl.Specs {
			tspec := spec.(*ast.TypeSpec)
			stCheck[tspec.Name.String()] = struct{}{}
			// check is generate struct by config
			if !IsGenerateStruct(tspec.Name.String()) {
				continue
			}
			st := &StructInfo{
				Typ: tspec.Name.String(),
			}
			if decl.Doc != nil {
				st.Doc = decl.Doc.Text()
			}
			if tspec.Comment != nil {
				st.Comment = tspec.Comment.Text()
			}
			data[tspec.Name.String()] = st
			// Composite structure check
			sti, ok := tspec.Type.(*ast.StructType)
			if !ok || sti.Fields == nil {
				return true
			}

			for _, field := range sti.Fields.List {
				// not compisite field
				if len(field.Names) > 0 {
					continue
				}
				var ident *ast.Ident
				switch fv := field.Type.(type) {
				case *ast.Ident:
					// struct composite
					ident = fv
				case *ast.StarExpr:
					// pointer composite
					switch rv := fv.X.(type) {
					case *ast.Ident:
						ident = rv
					case *ast.SelectorExpr:
						// // FIXME: composite other package
						// util.Dump(rv.Sel.Obj, "**field.select.Obj")
					}
				case *ast.SelectorExpr:
					// FIXME: composite other package
					// fmt.Println("-- ---->", fv.Sel.String(), fv.X.(*ast.Ident).String())
					// util.Dump(fv.Sel.Obj, "field.select.Obj")
					// util.Dump(fv.X.(*ast.Ident).Obj, "field.select.X.Obj")
				}
				if ident == nil {
					continue
				}
				// util.Dump(ident.Obj, "ident.Obj")
				// util.Dump(ident, "ident")
				if ident.Obj == nil {
					// FIX: compose other struct pointer
					st.Composites = append(st.Composites, &CompositeStructInfo{
						Typ:      ident.Name,
						IsStruct: true,
					})
					continue
				}
				isInterface := false
				if td, ok := ident.Obj.Decl.(*ast.TypeSpec); ok {
					switch tvv := td.Type.(type) {
					case *ast.StructType:
					case *ast.InterfaceType:
						isInterface = true
						_ = tvv
						// util.Dump(tvv, "ident.Obj.Interface.Type")
					}
				}
				var ct string
				ct = ident.String()
				if isInterface {
					// composite interface
					if len(dstPackage) > 0 {
						ct = dstPackage + "." + ct
					}
				} else {
					// composite struct
					if _, ok := data[ct]; !ok {
						data[ct] = &StructInfo{
							Typ:           ct,
							compositeOnly: true,
						}
					}
				}
				st.Composites = append(st.Composites, &CompositeStructInfo{
					Typ:      ct,
					IsStruct: !isInterface,
				})
			}

			// util.Dump(sti)
			return true
		}
		return true
	})

	if len(data) < 1 {
		// no struct to genrate
		// log.Println("not match any struct generate.")
		return
	}
	//
	changeType := func(typ string) string {
		if _, ok := stCheck[typ]; ok {
			return dstPackage + "." + typ
		}
		return typ
	}
	if len(dstPackage) == 0 {
		changeType = nil
	}
	// range func declare
	pkg.FuncDecl(func(decl *ast.FuncDecl, cm ast.CommentMap) bool {
		// ignore not method
		if decl.Recv == nil {
			return true
		}
		name := goparse.Format(pkg.Fset(), decl.Recv.List[0].Type)
		name = strings.TrimPrefix(name, "*")
		// find type info
		info, ok := data[name]
		if !ok {
			return true
		}
		// check export method by config
		if !IsGenerateMethod(decl.Name.String()) {
			return true
		}

		method := &StructMethod{
			Name: decl.Name.String(),
		}
		if decl.Doc != nil {
			method.Doc = decl.Doc.Text()
		}
		for _, param := range decl.Type.Params.List {
			field := &StructField{}
			for _, name := range param.Names {
				field.Names = append(field.Names,
					goparse.Format(pkg.Fset(), name))
			}
			field.Type = goparse.Format(pkg.Fset(), param.Type)
			method.Params = append(method.Params, field)
		}
		pos := pkg.Fset().Position(decl.Pos())
		// fmt.Sprintf("%s:%d", pos.Filename, pos.Line)
		method.fileName = pos.Filename
		method.fileLine = pos.Line
		method.Params = ToFileds(pkg.Fset(), decl.Type.Params, changeType)
		method.Results = ToFileds(pkg.Fset(), decl.Type.Results, changeType)
		info.Methods = append(info.Methods, method)
		return true
	})
	return
}

func HasPrefix(val string, ct ...string) bool {
	for _, v := range ct {
		if strings.HasPrefix(val, v) {
			return true
		}
	}
	return false
}

func ToFileds(fset *token.FileSet, in *ast.FieldList, changeType func(string) string) (out []*StructField) {
	if in == nil {
		return
	}
	for _, item := range in.List {
		field := &StructField{}
		for _, name := range item.Names {
			field.Names = append(field.Names,
				goparse.Format(fset, name))
		}
		field.Type = goparse.Format(fset, item.Type)
		if changeType != nil {
			field.Type = formatType(field.Type, changeType)
		}

		out = append(out, field)
	}
	return
}

func formatType(typ string, changeType func(string) string) string {
	typ = strings.TrimSpace(typ)
	if strings.HasPrefix(typ, "*") {
		return "*" + formatType(strings.TrimPrefix(typ, "*"), changeType)
	}
	if strings.HasPrefix(typ, "[]") {
		return "[]" + formatType(strings.TrimPrefix(typ, "[]"), changeType)
	}
	if strings.HasPrefix(typ, "map[") {
		typ = strings.TrimPrefix(typ, "map[")
		for k, v := range []byte(typ) {
			if v == ']' {
				key := string(typ[:k])
				val := string(typ[k+1:])
				return "map[" + formatType(key, changeType) + "]" + formatType(val, changeType)
			}
		}
		// invalid map define
		log.Fatalf("type map[%s split and format failed.", typ)
		return ""
	}
	if strings.HasPrefix(typ, "chan ") {
		return "chan " + formatType(strings.TrimPrefix(typ, "chan "), changeType)
	}
	return changeType(typ)
}
