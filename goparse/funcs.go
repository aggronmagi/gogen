package goparse

import (
	"bytes"
	"go/printer"
	"go/token"
)

// Format printer.Fprint wrap.
func Format(fset *token.FileSet, node interface{}) string {
	buf := &bytes.Buffer{}
	_ = printer.Fprint(buf, fset, node)
	return buf.String()
}
