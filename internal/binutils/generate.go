// +build ignore

package main

import (
	"net/http"
	"os"
	"path/filepath"

	"github.com/shurcooL/vfsgen"
	"github.com/sirupsen/logrus"
)

func main() {
	wd, err := os.Getwd()
	if err != nil {
		logrus.Fatal(err)
	}
	assets := http.Dir(filepath.Join(wd, "server/static"))
	if err := vfsgen.Generate(assets, vfsgen.Options{
		Filename:     filepath.Join(wd, "internal/binutils", "static.go"),
		PackageName:  "binutils",
		VariableName: "Assets",
	}); err != nil {
		logrus.Fatal(err)
	}
}
