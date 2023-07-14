package logx

import (
	"path/filepath"
	"strings"

	"github.com/aggronmagi/gogen/goparse"
	"github.com/aggronmagi/gogen/internal/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var cfg = struct {
	BuildTags []string
	DstPkg    string
	pkgName   string
	Replace   bool
	Stdout    bool
}{
	DstPkg: "xxxx.com/xx/logx",
}

func FlagSet(set *pflag.FlagSet) {
	set.StringSliceVar(&cfg.BuildTags, "tags", cfg.BuildTags, "comma-separated list of build tags to apply")
	set.StringVarP(&cfg.DstPkg, "dst-package", "p", cfg.DstPkg, "replace logx package")
	set.BoolVarP(&cfg.Replace, "replace", "r", cfg.Replace, "replace logx action,if not set, fix zap logger")
	set.BoolVar(&cfg.Stdout, "stdout", cfg.Stdout, "output to stdout")
}

// Version option command version
var Version string = "0.0.1"

func RunCommand(cmd *cobra.Command, args ...string) {
	if len(args) < 1 {
		cmd.Help()
		return
	}
	pkgs, err := goparse.ParseMulPackage(args, cfg.BuildTags...)
	util.FatalIfErr(err, "parse failed")

	cfg.DstPkg = strings.TrimSpace(cfg.DstPkg)

	cfg.pkgName = filepath.Base(cfg.DstPkg)

	for _, pkg := range pkgs {
		for _, file := range pkg.Syntax {
			if cfg.Replace {
				ReplaceZapLogger(file, pkg.Fset)
			} else {
				FixZapLogger(file, pkg.Fset)
			}
		}
	}
}
