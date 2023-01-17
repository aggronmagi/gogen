package template

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// command config
var config = struct {
	TypeNames   []string
	Output      string
	TrimPrefix  string
	LineComment bool
	BuildTags   []string
}{
	TypeNames:   []string{},
	Output:      "",
	TrimPrefix:  "",
	LineComment: false,
	BuildTags:   []string{},
}

// Version generate tool version
var Version string = "0.0.1"

// Flags generate tool flags
func Flags(set *pflag.FlagSet) {
	set.StringSliceVarP(&config.TypeNames, "type", "t", config.TypeNames, "list of type names; must be set")
	set.StringVarP(&config.Output, "output", "o", config.Output, "output file name; default srcdir/<type>_string.go")
	set.StringVarP(&config.TrimPrefix, "trimprefix", "p", config.TrimPrefix, "trim the `prefix` from the generated constant names")
	set.BoolVar(&config.LineComment, "linecomment", false, "use line comment text as printed text when present")
	set.StringSliceVar(&config.BuildTags, "tags", config.BuildTags, "comma-separated list of build tags to apply")
}

// RunCommand run generate command
func RunCommand(cmd *cobra.Command, args []string) {

	if len(config.TypeNames) < 1 {
		log.Println("not set -t or --type")
		cmd.Help()
		os.Exit(2)
	}
}
