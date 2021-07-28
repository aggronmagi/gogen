// Package gen provide tools for generate go files.
package gen

import (
	"bytes"
	"fmt"
	"go/format"
	"io/ioutil"
	"log"
	"os/exec"
	"strings"
)

// Generator used to buffer the output for format.Source.
type Generator struct {
	Buf          bytes.Buffer // Accumulated output.
	FormatSource func(in []byte) (out []byte, err error)
}

func (g *Generator) Print(args ...interface{}) {
	fmt.Fprint(&g.Buf, args...)
}

func (g *Generator) Printf(format string, args ...interface{}) {
	fmt.Fprintf(&g.Buf, format, args...)
}

func (g *Generator) Println(args ...interface{}) {
	fmt.Fprintln(&g.Buf, args...)
}

func (g *Generator) PrintDoc(docs string) {
	docs = strings.TrimSpace(docs)
	if len(docs) < 1 {
		return
	}
	list := strings.Split(docs, "\n")
	for _, v := range list {
		g.Printf("// %s\n", strings.TrimSpace(v))
	}
}

// format returns the gofmt-ed contents of the Generator's buffer.

func (g *Generator) Write(file string) (err error) {
	fmtsrc := g.FormatSource
	if fmtsrc == nil {
		fmtsrc = DefaultFormat
	}
	src, err := fmtsrc(g.Buf.Bytes())
	if err != nil {
		// Should never happen, but can arise when developing this code.
		// The user can compile the output to see the error.
		log.Printf("warning: internal error: invalid Go generated: %s", err)
		log.Printf("warning: compile the package to analyze the error")
		src = g.Buf.Bytes()
	}

	return ioutil.WriteFile(file, src, 0644)
}

// EmptyFormat do not format
func EmptyFormat(in []byte) ([]byte, error) {
	return in, nil
}

// DefaultFormat default format source
func DefaultFormat(in []byte) (out []byte, err error) {
	return format.Source(in)
}

// OptionGoimportsFormtat use the command line `goimports` command to
// format first. If an error is reported, use the default formatting method.
func OptionGoimportsFormtat(in []byte) ([]byte, error) {
	out := bytes.NewBuffer(nil)
	cmd := exec.Command("goimports")
	cmd.Stdin = bytes.NewBuffer(in)
	cmd.Stdout = out
	err := cmd.Run()
	if err == nil {
		return out.Bytes(), nil
	}
	return format.Source(in)
}

func SprintDoc(in string) (out string) {
	in = strings.TrimSpace(in)
	if len(in) < 1 {
		return
	}
	list := strings.Split(in, "\n")
	for _, v := range list {
		out += "// "
		out += strings.TrimSpace(v)
		out += "\n"
	}
	return
}
