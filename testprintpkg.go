package main

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	"io"
	"os"
	"reflect"

	"golang.org/x/tools/go/packages"
)

const usage = `testprintpkg - print AST to stdout, defs and uses to stderr
Usage:
  testprintpkg PATTERNS...
    e.g. testprintpkg ./...`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(1)
	}
	pkgs, err := packages.Load(&packages.Config{
		Mode: packages.NeedTypesInfo | packages.NeedTypes | packages.NeedName | packages.NeedSyntax | packages.NeedImports,
	}, os.Args[1:]...)
	fail(err)

	pkgSet := make(map[string]*packages.Package, len(pkgs))
	for _, pkg := range pkgs {
		pkgSet[pkg.Name] = pkg
	}

	for _, pkg := range pkgs {
		fail(printAST(os.Stdout, pkg))
		lgr := &logger{
			pkg:    pkg,
			pkgSet: pkgSet,
			w:      os.Stderr,
		}
		lgr.printDefs()
		lgr.printUses()
	}
}

func fail(err error) {
	if err != nil {
		panic(err)
	}
}

func printAST(w io.Writer, pkg *packages.Package) error {
	return ast.Fprint(w, pkg.Fset, pkg, ast.NotNilFilter)
}

type logger struct {
	prefix string
	pkg    *packages.Package
	pkgSet map[string]*packages.Package
	w      io.Writer
}

func (s *logger) setPrefix(prefix string) { s.prefix = fmt.Sprintf("[%s][%s]", s.pkg.Name, prefix) }
func (s *logger) writef(format string, v ...interface{}) {
	fmt.Fprintf(s.w, "%s %s\n", s.prefix, fmt.Sprintf(format, v...))
}

func (s *logger) printUses() {
	s.setPrefix("uses")
	s.printObjects(s.pkg.TypesInfo.Uses)
}

func (s *logger) printDefs() {
	s.setPrefix("defs")
	s.printObjects(s.pkg.TypesInfo.Defs)
}

func (s *logger) printObjects(objMap map[*ast.Ident]types.Object) {
	for ident, obj := range objMap {
		identPos := s.pkg.Fset.Position(ident.Pos())
		if obj == nil {
			s.writef("%v (nil)", identPos)
			continue
		}
		pkgName := func() string {
			if obj.Pkg() == nil {
				return "builtin"
			}
			return obj.Pkg().Name()
		}()
		objType := reflect.TypeOf(obj.Type())
		if ident.Pos() == obj.Pos() {
			s.writef("%v (%d) %s %s %s", identPos, ident.Pos(), pkgName, types.ObjectString(obj, nil), objType)
			continue
		}
		s.writef("%v (%d) => %s %v (%d) %s %s",
			identPos, ident.Pos(), pkgName, s.objPos(obj), obj.Pos(), types.ObjectString(obj, nil), objType)
	}
}

func (s *logger) objPos(obj types.Object) token.Position {
	if obj.Pkg() == nil {
		return token.Position{
			Filename: "unknown",
		}
	}
	if p, ok := s.pkgSet[obj.Pkg().Name()]; ok {
		return p.Fset.Position(obj.Pos())
	}
	return token.Position{
		Filename: obj.Pkg().Name(),
	}
}
