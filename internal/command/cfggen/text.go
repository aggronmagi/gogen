package cfggen

var configTemplate = `// Code generated by "gogen cfggen"; DO NOT EDIT.
// Exec: gogen {{.Commands}} Version: {{.Version}}
package {{.PackageName}}

import (
    "time"
    "github.com/spf13/viper"
    "github.com/walleframe/walle/services/configcentra"
)

var _ = {{.FromFunc}}()

// {{.Name}} config generate by gogen cfggen.
type {{.Name}} struct { {{- range $i,$f := .Fields }}
    {{Comment $f -}}
    {{$f.Name}} {{$f.Type}} {{Tag "json" $f.Name "omitempty" }} {{- end }}
    // config prefix string
    prefix string
    // update ntf funcs
    ntfFuncs []func(*{{.Name}})
}

var _ configcentra.ConfigValue = (*{{.Name}})(nil)

func New{{.Name}}(prefix string) *{{.Name}}{
	if prefix == "" {
		panic("config prefix invalid")
	}
    // new default config value
    cfg := NewDefault{{.Name}}(prefix)
    // register value to config centra
    configcentra.RegisterConfig(cfg)
    return cfg
}

func NewDefault{{.Name}}(prefix string)*{{.Name}}{
    cfg := &{{.Name -}} { {{- range $i,$f := .Fields}}
        {{$f.Name}} : {{$f.Body}}, {{- end}}
        prefix : prefix,
    }
    return cfg
}

// add notify func
func (cfg *{{.Name}}) AddNotifyFunc(f func(*{{.Name}})) {
    cfg.ntfFuncs = append(cfg.ntfFuncs, f)
}

// impl configcentra.ConfigValue
func (cfg *{{.Name}}) SetDefaultValue(cc configcentra.ConfigCentra) {
	if cc.UseObject() {
		cc.SetObject(cfg.prefix, "{{Doc .Document .Comment}}", cfg)
		return 
	}
{{- range $i,$f := .Fields}}
    cc.SetDefault(cfg.prefix + ".{{ToLower $f.Name}}", "{{OneRow $f.Doc}}", cfg.{{$f.Name}}) {{- end }} 
}

// impl configcentra.ConfigValue
func (cfg *{{.Name}}) RefreshValue(cc configcentra.ConfigCentra) error {
    if cc.UseObject()  {
		return cc.GetObject(cfg.prefix, cfg)
    } {{- range $i,$f := .Fields}}
    {
		v,err := cc.{{$f.GetMethod}}(cfg.prefix + ".{{ToLower $f.Name}}")
		if err != nil {
			return err
		}
		cfg.{{$f.Name}} = ({{$f.Type}})(v)
	} {{- end }}
    // notify update
    for _, ntf := range cfg.ntfFuncs {
        ntf(cfg)
    }
	return nil
}

`