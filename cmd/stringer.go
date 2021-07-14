/*
Copyright © 2021 chenzhiyuan

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/
package cmd

import (
	"github.com/aggronmagi/gogen/internal/command/stringer"
	"github.com/spf13/cobra"
)

// stringerCmd represents the stringer command
var stringerCmd = &cobra.Command{
	Use:   "stringer [flags] -t T [directory | files ]",
	Short: "Stringer automate create fmt.Stringer interface",
	Long: `Stringer automate create fmt.Stringer interface

Copyright 2014 The Go Authors. All rights reserved.

Stringer is a tool to automate the creation of methods that satisfy the fmt.Stringer
interface. Given the name of a (signed or unsigned) integer type T that has constants
defined, stringer will create a new self-contained Go source file implementing
	func (t T) String() string
The file is created in the same package and directory as the package that defines T.
It has helpful defaults designed for use with go generate.

For more information, see:
	https://pkg.go.dev/golang.org/x/tools/cmd/stringer
`,
	Run: func(cmd *cobra.Command, args []string) {
		stringer.RunCommand(cmd, args)
	},
}

func init() {
	rootCmd.AddCommand(stringerCmd)

	stringer.Flags(stringerCmd.Flags())

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// stringerCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// stringerCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
