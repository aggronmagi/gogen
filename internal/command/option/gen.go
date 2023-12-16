package option

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/aggronmagi/gogen/gen"
	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
)

func generate(pkg *goparse.Package, st *optionStruct) {

	var data = struct {
		*optionStruct
		ExecArgs    string
		Version     string
		PackageName string
		GenAppend   bool
	}{
		optionStruct: st,
		ExecArgs:     strings.Join(os.Args[1:], " "),
		Version:      Version,
		PackageName:  pkg.Package().Name,
		GenAppend:    config.GenAppend,
	}

	tpl := template.New(config.Template).Funcs(UseFuncMap)

	importCache := make([]struct {
		pkg   string
		alias string
	}, 0, 8)
	importFunc := func(pkg string, alias ...string) (_ string) {
		for _, v := range importCache {
			if pkg == v.pkg {
				return
			}
		}

		rename := ""
		if len(alias) >= 0 {
			rename = alias[0]
			if rename == filepath.Base(pkg) {
				rename = ""
			}
		}

		importCache = append(importCache, struct {
			pkg   string
			alias string
		}{
			pkg:   pkg,
			alias: rename,
		})
		return
	}

	customImport := func() string {
		buf := strings.Builder{}
		for _, v := range importCache {
			buf.WriteByte('\t')
			buf.WriteString(v.alias)
			buf.WriteString(` "`)
			buf.WriteString(v.pkg)
			buf.WriteByte('"')
			buf.WriteByte('\n')
		}
		return buf.String()
	}

	tpl.Funcs(template.FuncMap{
		"Import": importFunc,
	})

	var err error
	switch config.Template {
	case "option":
		_, err = tpl.Parse(tplOption)
	// case "config":
	// 	_, err = tpl.Parse(tplConfig)
	default:
		if config.Template == "" {
			log.Println("invalid template config")
			return
		}
		var data []byte
		data, err = os.ReadFile(config.Template)
		if err != nil {
			log.Println("load config file failed,", err)
			return
		}

		_, err = tpl.Parse(string(data))
	}

	if err != nil {
		log.Println("parse template failed,", err)
		return
	}

	for _, v := range pkg.Package().Imports {
		importFunc(v.PkgPath, v.Name)
	}

	buf := &bytes.Buffer{}
	err = tpl.Execute(buf, data)
	if err != nil {
		log.Println("execute template failed,", err)
		return
	}
	file := config.Output
	if file == "" {
		file = "gen_" + strings.ToLower(st.Name) + ".go"
	}

	bdata := bytes.Replace(buf.Bytes(), []byte("$Import-Package$"), []byte(fmt.Sprintf("import (\n%s)", customImport())), 1)
	g := &gen.Generator{
		FormatSource: gen.OptionGoimportsFormtat,
		Buf:          *bytes.NewBuffer(bdata),
	}
	err = g.Write(file)
	util.FatalIfErr(err, "save output failed")
}
