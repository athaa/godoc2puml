package parser

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/build"
	"go/parser"
	"go/token"
	"os"
	"strings"
	. "github.com/t-yuki/godoc2puml/ast"
)

func ParsePackage(packagePath string) (*Package, error) {
	p := &Package{}
	p.QualifiedName = packagePath
	p.Classes = make([]Class, 0, 10)

	buildPkg, err := build.Import(packagePath, ".", build.FindOnly)
	if err != nil {
		return nil, err
	}
	dir := buildPkg.Dir

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, dir, func(fi os.FileInfo) bool {
		return !fi.IsDir() && !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		return nil, err
	}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			parseFile(p, file)
		}
	}
	return p, nil
}

func parseFile(p *Package, f *ast.File) {
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		for _, spec := range gd.Specs {
			ts, ok1 := spec.(*ast.TypeSpec)
			if !ok1 {
				continue
			}

			st, ok2 := ts.Type.(*ast.StructType)
			if !ok2 {
				continue
			}

			cl := Class{Name: ts.Name.Name, Relations: make([]Relation, 0, 10)}
			parseFields(&cl, st.Fields)
			p.Classes = append(p.Classes, cl)
		}
	}
}

func parseFields(cl *Class, fields *ast.FieldList) {
	for _, field := range fields.List {
		multiplicity := ""
		if _, ok := field.Type.(*ast.ArrayType); ok {
			multiplicity = "0..*"
		}
		elementType := elementType(field.Type)
		switch {
		case isPrimitive(elementType):
			f := Field{Type: elementType, Multiplicity: multiplicity}

			if len(field.Names) == 0 { // anonymous field
				cl.Fields = append(cl.Fields, f)
			}
			for _, name := range field.Names {
				f.Name = name.String()
				cl.Fields = append(cl.Fields, f)
			}
		default:
			rel := Relation{Target: elementType, Multiplicity: multiplicity}

			if len(field.Names) == 0 { // anonymous field
				rel.RelType = Composition
				cl.Relations = append(cl.Relations, rel)
			}
			for _, name := range field.Names {
				rel.Label = name.String()
				rel.RelType = Association
				cl.Relations = append(cl.Relations, rel)
			}
		}
	}
}

func elementType(expr ast.Node) string {
	if expr == nil {
		return ""
	}
	switch expr := expr.(type) {
	case *ast.Ident:
		return expr.String()
	case *ast.ArrayType:
		return elementType(expr.Elt)
	case *ast.StarExpr:
		return elementType(expr.X)
	case *ast.SelectorExpr:
		return elementType(expr.X) + "." + expr.Sel.String()
	case *ast.FuncType:
		return "func " + elementType(expr.Params) + elementType(expr.Results)
	case *ast.FieldList:
		if expr == nil {
			return ""
		}
		var buf bytes.Buffer
		for _, field := range expr.List {
			buf.WriteString(elementType(field.Type))
		}
		return buf.String()
	case *ast.MapType:
		return "map[" + elementType(expr.Key) + "]" + elementType(expr.Value)
	case *ast.InterfaceType:
		return "interface {" + elementType(expr.Methods) + "}"
	case *ast.StructType:
		return "struct {" + elementType(expr.Fields) + "}"
	case *ast.ChanType:
		switch expr.Dir {
		case ast.SEND:
			return "chan out " + elementType(expr.Value)
		case ast.RECV:
			return "chan in " + elementType(expr.Value)
		default:
			return "chan both " + elementType(expr.Value)
		}
	default:
		panic(fmt.Errorf("%#v", expr))
	}
}

func isPrimitive(name string) bool {
	switch name {
	case "bool", "int", "uint", "byte", "float",
		"uint8", "int8",
		"uint16", "int16",
		"uint32", "int32",
		"uint64", "int64",
		"string":
		return true
	default:
	}
	switch {
	case strings.ContainsAny(name, " ["):
		return true
	default:
		return false
	}
}
