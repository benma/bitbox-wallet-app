package mobileserver

import (
	"fmt"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func TestPackage(t *testing.T) {
	cfg := &packages.Config{Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo}
	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		panic(err)
	}
	if len(pkgs) == 0 {
		panic("package not found")
	}
	var pkg *types.Package = pkgs[0].Types
	for _, name := range pkg.Scope().Names() {
		fmt.Println(name, pkg.Scope().Lookup(name).Exported())
	}
}
