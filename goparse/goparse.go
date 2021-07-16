// Package goparse is the help package of Go language ast tree.
package goparse

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Package struct {
	pkg *packages.Package
}

// ParseGeneratePackage by go generate env
func ParseGeneratePackage(tags ...string) (pkg *Package, err error) {
	cfg := &packages.Config{
		Mode:       packages.LoadSyntax,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}
	pkgs, err := packages.Load(cfg, EnvGoFile)
	if err != nil {
		return
	}
	if len(pkgs) != 1 {
		err = fmt.Errorf("error: %d packages found", len(pkgs))
		return
	}
	pkg = &Package{
		pkg: pkgs[0],
	}

	return
}

func ParsePackage(patterns []string, tags ...string) (pkg *Package, err error) {
	cfg := &packages.Config{
		Mode:       packages.LoadSyntax,
		Tests:      false,
		BuildFlags: []string{fmt.Sprintf("-tags=%s", strings.Join(tags, " "))},
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return
	}
	if len(pkgs) != 1 {
		err = fmt.Errorf("error: %d packages found", len(pkgs))
		return
	}
	pkg = &Package{
		pkg: pkgs[0],
	}
	return
}

func (p *Package) GenDecl(f func(decl *ast.GenDecl) bool) {
	for _, file := range p.pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if decl, ok := n.(*ast.GenDecl); ok {
				return f(decl)
			}
			return true
		})
	}
}

func (p *Package) FuncDecl(f func(decl *ast.FuncDecl) bool) {
	for _, file := range p.pkg.Syntax {
		ast.Inspect(file, func(n ast.Node) bool {
			if decl, ok := n.(*ast.FuncDecl); ok {
				return f(decl)
			}
			return true
		})
	}
}

func (p *Package) ConstDecl(f func(decl *ast.GenDecl, vspec *ast.ValueSpec) bool) {
	p.GenDecl(func(decl *ast.GenDecl) bool {
		if decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vspec := spec.(*ast.ValueSpec)
			if !f(decl, vspec) {
				return false
			}
		}
		return true
	})
}

func (p *Package) ConstDeclWithType(t string, f func(decl *ast.GenDecl, vspec *ast.ValueSpec) bool) {
	p.GenDecl(func(decl *ast.GenDecl) bool {
		if decl.Tok != token.CONST {
			return true
		}
		typ := ""
		for _, spec := range decl.Specs {
			vspec := spec.(*ast.ValueSpec)

			if vspec.Type == nil && len(vspec.Values) > 0 {
				// "X = 1". With no type but a value. If the constant is untyped,
				// skip this vspec and reset the remembered type.
				typ = ""

				// If this is a simple type conversion, remember the type.
				// We don't mind if this is actually a call; a qualified call won't
				// be matched (that will be SelectorExpr, not Ident), and only unusual
				// situations will result in a function call that appears to be
				// a type conversion.
				ce, ok := vspec.Values[0].(*ast.CallExpr)
				if !ok {
					continue
				}
				id, ok := ce.Fun.(*ast.Ident)
				if !ok {
					continue
				}
				typ = id.Name
			}
			if vspec.Type != nil {
				// "X T". We have a type. Remember it.
				ident, ok := vspec.Type.(*ast.Ident)
				if !ok {
					continue
				}
				typ = ident.Name
			}
			if typ != t {
				// This is not the type we're looking for.
				continue
			}
			if !f(decl, vspec) {
				return false
			}
		}
		return false
	})
}

func (p *Package) TypeDecl(f func(decl *ast.GenDecl, typ *ast.TypeSpec) bool) {
	p.GenDecl(func(decl *ast.GenDecl) bool {
		if decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			tspec := spec.(*ast.TypeSpec)
			if !f(decl, tspec) {
				return false
			}
		}
		return true
	})
}

func (p *Package) TypeDeclWithName(typ string, f func(decl *ast.GenDecl, typ *ast.TypeSpec)) {
	p.GenDecl(func(decl *ast.GenDecl) bool {
		if decl.Tok != token.TYPE {
			return true
		}

		for _, spec := range decl.Specs {
			tspec := spec.(*ast.TypeSpec)
			if tspec.Name.String() != typ {
				continue
			}
			f(decl, tspec)
			return false
		}
		return true
	})
}

func (p *Package) VarDecl(f func(decl *ast.GenDecl, typ *ast.ValueSpec) bool) {
	p.GenDecl(func(decl *ast.GenDecl) bool {
		if decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			vspec := spec.(*ast.ValueSpec)
			if !f(decl, vspec) {
				return false
			}
		}
		return true
	})
}

func (p *Package) GetDefObj(name *ast.Ident) (obj types.Object, ok bool) {
	obj, ok = p.pkg.TypesInfo.Defs[name]
	return
}

func (p *Package) Position(pos token.Pos) token.Position {
	return p.pkg.Fset.Position(pos)
}

func (p *Package) Package() *packages.Package {
	return p.pkg
}

func (p *Package) Fset() *token.FileSet {
	return p.pkg.Fset
}

func (p *Package) GetGenerateNode() (node ast.Node) {
	for k, name := range p.pkg.CompiledGoFiles {
		if name != EnvGoFile {
			continue
		}
		file := p.pkg.Syntax[k]
		search := func(lineAdd int) {
			ast.Inspect(file, func(n ast.Node) bool {
				if p.Position(n.Pos()).Line == EnvGoLine+lineAdd {
					node = n
					return false
				}
				return true
			})
		}
		// add compatibility
		for add := 1; add < 3; add++ {
			search(add)
			if node != nil {
				return
			}
		}

		break
	}
	return
}
