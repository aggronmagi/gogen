package goparse

import (
	"os"
	"strconv"
)

// go generate tools environment value.
const (
	GOARCH    = "GOARCH"
	GOOS      = "GOOS"
	GOPATH    = "GOPATH"
	GOPACKAGE = "GOPACKAGE"
	GOFILE    = "GOFILE"
	GOLINE    = "GOLINE"

	GoGeneratePrefix = "//go:generate"
)

// go generate tools environment value.
var (
	EnvGoArch    string
	EnvGoOS      string
	EnvGoPath    string
	EnvGoPackage string
	EnvGoFile    string
	EnvGoLine    int
)

// parse tools env values
func init() {
	EnvGoArch = os.Getenv(GOARCH)
	EnvGoOS = os.Getenv(GOOS)
	EnvGoPath = os.Getenv(GOPATH)
	EnvGoPackage = os.Getenv(GOPACKAGE)
	EnvGoFile = os.Getenv(GOFILE)
	line := os.Getenv(GOLINE)
	EnvGoLine, _ = strconv.Atoi(line)
}
