package tgoimporter

import (
	"sync"

	"github.com/tgo-lang/lang/ast"
	"github.com/tgo-lang/lang/parser"
	"github.com/tgo-lang/lang/token"
	"github.com/tgo-lang/lang/types"
)

type TgoDefaultImporter struct {
	I types.ImporterFrom
}

func (f *TgoDefaultImporter) Import(path string) (*types.Package, error) {
	if path == "github.com/mateusz834/tgo" {
		return tgoPkg()
	}
	return f.I.Import(path)
}

func (f *TgoDefaultImporter) ImportFrom(path, dir string, mode types.ImportMode) (*types.Package, error) {
	if path == "github.com/mateusz834/tgo" {
		return tgoPkg()
	}
	return f.I.ImportFrom(path, dir, mode)
}

// TODO: test that proves (only for CI) that this is the same as in the tgo repo.
var tgoPkg = sync.OnceValues(func() (*types.Package, error) {
	const tgoModuleSrc = `package tgo
type Ctx struct{}
type Error = error
type UnsafeHTML string
type DynamicWriteAllowed interface {
	string|UnsafeHTML|int|uint|rune
}
func DynamicWrite[T DynamicWriteAllowed](t T) {
}
`
	fset := token.NewFileSet()
	tgoModuleFile, err := parser.ParseFile(fset, "tgo.go", tgoModuleSrc, parser.SkipObjectResolution)
	if err != nil {
		return nil, err
	}

	tgoPkg, err := new(types.Config).Check("github.com/tgo-lang/tgo", fset, []*ast.File{tgoModuleFile}, nil)
	if err != nil {
		return nil, err
	}

	return tgoPkg, nil
})
