package cfggen

import (
	"bytes"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"text/template"

	"github.com/aggronmagi/gogen/gen"
	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"golang.org/x/tools/go/packages"
)

func generate(pkg *goparse.Package, st *optionStruct) {

	value := struct {
		Commands    string
		Version     string
		PackageName string
		*optionStruct
		Imports map[string]*packages.Package
	}{
		Commands:     strings.Join(os.Args[1:], " "),
		Version:      Version,
		PackageName:  pkg.Package().Name,
		optionStruct: st,
		Imports:      pkg.Package().Imports,
	}

	tmpl, err := template.New("cfggen").Funcs(UseFuncMap).Parse(configTemplate)
	if err != nil {
		log.Fatal(err)
	}
	var buf = bytes.NewBuffer(nil)
	err = tmpl.Execute(buf, value)
	if err != nil {
		log.Fatal("exec template failed.", err)
	}
	fd, err := gen.OptionGoimportsFormtat(buf.Bytes())
	if err != nil {
		log.Println(buf.String())
		log.Fatal("format go code failed", err)
	}
	file := config.Output
	if file == "" {
		file = "gen_" + strings.ToLower(st.Name) + ".go"
	}

	err = ioutil.WriteFile(file, fd, 0644)
	log.Println("==>", config.Output)
	util.FatalIfErr(err, "format and write result")
}
