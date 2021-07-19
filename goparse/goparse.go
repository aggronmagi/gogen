// Package goparse is the help package of Go language ast tree.
package goparse

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/packages"
)

type Package struct {
	pkg *packages.Package
}

// ParsePackage parse specified packages.
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

// GenDecl range all GenDecl
func (p *Package) GenDecl(f func(decl *ast.GenDecl, cm ast.CommentMap) bool) {
	for _, file := range p.pkg.Syntax {
		cm := ast.NewCommentMap(p.pkg.Fset, file, file.Comments)
		ast.Inspect(file, func(n ast.Node) bool {
			if decl, ok := n.(*ast.GenDecl); ok {
				return f(decl, cm)
			}
			return true
		})
	}
}

// FuncDecl range func declare
func (p *Package) FuncDecl(f func(decl *ast.FuncDecl, cm ast.CommentMap) bool) {
	for _, file := range p.pkg.Syntax {
		cm := ast.NewCommentMap(p.pkg.Fset, file, file.Comments)
		ast.Inspect(file, func(n ast.Node) bool {
			if decl, ok := n.(*ast.FuncDecl); ok {
				return f(decl, cm)
			}
			return true
		})
	}
}

// ConstDecl range const value
func (p *Package) ConstDeclValue(f func(decl *ast.GenDecl, vspec *ast.ValueSpec, cm ast.CommentMap) bool) {
	p.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
		if decl.Tok != token.CONST {
			return true
		}
		for _, spec := range decl.Specs {
			vspec := spec.(*ast.ValueSpec)
			if !f(decl, vspec, cm) {
				return false
			}
		}
		return true
	})
}

// ConstDeclValueWithType range specified type const values
func (p *Package) ConstDeclValueWithType(t string, f func(decl *ast.GenDecl, vspec *ast.ValueSpec, cm ast.CommentMap) bool) {
	p.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
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
			if !f(decl, vspec, cm) {
				return false
			}
		}
		return false
	})
}

// TypeDecl range type declare
func (p *Package) TypeDecl(f func(decl *ast.GenDecl, typ *ast.TypeSpec, cm ast.CommentMap) bool) {
	p.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
		if decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			tspec := spec.(*ast.TypeSpec)
			if !f(decl, tspec, cm) {
				return false
			}
		}
		return true
	})
}

// TypeDeclWithName run with specified type define
func (p *Package) TypeDeclWithName(typ string, f func(decl *ast.GenDecl, typ *ast.TypeSpec, cm ast.CommentMap)) {
	p.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
		if decl.Tok != token.TYPE {
			return true
		}

		for _, spec := range decl.Specs {
			tspec := spec.(*ast.TypeSpec)
			if tspec.Name.String() != typ {
				continue
			}
			f(decl, tspec, cm)
			return false
		}
		return true
	})
}

// VarDecl range value define
func (p *Package) VarDecl(f func(decl *ast.GenDecl, typ *ast.ValueSpec, cm ast.CommentMap) bool) {
	p.GenDecl(func(decl *ast.GenDecl, cm ast.CommentMap) bool {
		if decl.Tok != token.TYPE {
			return true
		}
		for _, spec := range decl.Specs {
			vspec := spec.(*ast.ValueSpec)
			if !f(decl, vspec, cm) {
				return false
			}
		}
		return true
	})
}

// GetDefObj get define object
func (p *Package) GetDefObj(name *ast.Ident) (obj types.Object, ok bool) {
	obj, ok = p.pkg.TypesInfo.Defs[name]
	return
}

// Position transport to Position
func (p *Package) Position(pos token.Pos) token.Position {
	return p.pkg.Fset.Position(pos)
}

// Package get original package
func (p *Package) Package() *packages.Package {
	return p.pkg
}

// Fset get fileset
func (p *Package) Fset() *token.FileSet {
	return p.pkg.Fset
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

// GetGenerateNode get go generate tools specified node
func (p *Package) GetGenerateNode() (node ast.Node, cm ast.CommentMap, err error) {
	dir, err := os.Getwd()
	if err != nil {
		err = fmt.Errorf("get current directory: %w", err)
	}
	for k, name := range p.pkg.CompiledGoFiles {
		fname := filepath.Join(dir, EnvGoFile)
		if fname != name {
			continue
		}
		file := p.pkg.Syntax[k]
		cm = ast.NewCommentMap(p.pkg.Fset, file, file.Comments)
		search := func(lineAdd int) {
			ast.Inspect(file, func(n ast.Node) bool {
				if n == nil {
					return true
				}
				if p.Position(n.Pos()).Line == EnvGoLine+lineAdd {
					node = n
					return false
				}
				return true
			})
		}
		// add compatibility
		for add := 1; add < 1+GoGenerateToolsCompatibilityLineCount; add++ {
			search(add)
			if node != nil {
				return
			}
		}

		break
	}
	if node == nil {
		err = fmt.Errorf("not found %s:%d ", EnvGoFile, EnvGoLine)
	}
	return
}

// GoGenerateToolsCompatibilityLineCount use to modify compatibility line count
var GoGenerateToolsCompatibilityLineCount int = 2
