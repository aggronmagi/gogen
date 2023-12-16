package option

import (
	"strings"
	"text/template"
)

var UseFuncMap = template.FuncMap{}

func init() {
	UseFuncMap["Doc"] = funDoc
	UseFuncMap["TailDoc"] = func(docs []string) string {
		if len(docs) < 1 {
			return ""
		}
		return funDoc(docs[0])
	}
	UseFuncMap["Title"] = func(in string) (out string) {
		list := strings.Split(in, "_")
		for _, v := range list {
			out += strings.Title(v)
		}
		return
	}
}

func funDoc(docs string) string {
	docs = strings.TrimSpace(docs)
	if len(docs) < 1 {
		return ""
	}
	list := strings.Split(docs, "\n")
	buf := strings.Builder{}
	for _, v := range list {
		buf.WriteString("// ")
		buf.WriteString(strings.TrimSpace(v))
		buf.WriteByte('\n')
	}
	return buf.String()
}
