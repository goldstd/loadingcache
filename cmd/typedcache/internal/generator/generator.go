// Package generator logic is heavily inspired by https://github.com/vektah/dataloaden
package generator

import (
	"bytes"
	"fmt"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/imports"
)

type templateValues struct {
	Name      string
	KeyType   *goType
	ValueType *goType
	Package   string
}

type goType struct {
	Modifiers  string
	ImportPath string
	ImportName string
	Name       string
}

func (t *goType) String() string {
	if t.ImportName != "" {
		return t.Modifiers + t.ImportName + "." + t.Name
	}

	return t.Modifiers + t.Name
}

func (t *goType) IsPtr() bool {
	return strings.HasPrefix(t.Modifiers, "*")
}

func (t *goType) IsSlice() bool {
	return strings.HasPrefix(t.Modifiers, "[]")
}

var partsRe = regexp.MustCompile(`^([\[\]\*]*)(.*?)(\.\w*)?$`)
var tmpl = template.Must(template.New("typedCacheTemplate").Parse(typedCacheTemplate))

// parseType converts a string to a richer data structure.
// It is capable of parsing simple types, like "string" or
// imported types in the format github.com/import/path.Name.
func parseType(str string) (*goType, error) {
	parts := partsRe.FindStringSubmatch(str)
	if len(parts) != 4 {
		return nil, fmt.Errorf("type must be in the form []*github.com/import/path.Name")
	}

	t := &goType{
		Modifiers:  parts[1],
		ImportPath: parts[2],
		Name:       strings.TrimPrefix(parts[3], "."),
	}

	if t.Name == "" {
		t.Name = t.ImportPath
		t.ImportPath = ""
	}

	if t.ImportPath != "" {
		p, err := packages.Load(&packages.Config{Mode: packages.NeedName}, t.ImportPath)
		if err != nil {
			return nil, err
		}
		if len(p) != 1 {
			return nil, fmt.Errorf("not found")
		}

		t.ImportName = p[0].Name
	}

	return t, nil
}

func getPackage(dir string) (*packages.Package, error) {
	p, err := packages.Load(&packages.Config{
		Dir: dir,
	}, ".")

	if err != nil {
		return nil, err
	}

	if len(p) != 1 {
		return nil, fmt.Errorf("unable to find package info for %s", dir)
	}

	return p[0], nil
}

// Generate generates the template and writes it to disk
func Generate(wd string, name string, keyType string, valueType string) error {
	tmplValues := templateValues{Name: name}

	var err error
	tmplValues.KeyType, err = parseType(keyType)
	if err != nil {
		return err
	}
	tmplValues.ValueType, err = parseType(valueType)
	if err != nil {
		return err
	}

	genPkg, err := getPackage(wd)
	if err != nil {
		return err
	}
	tmplValues.Package = genPkg.Name

	// if we are inside the same package as the type we don't need an import and can refer directly to the type
	if genPkg.PkgPath == tmplValues.ValueType.ImportPath {
		tmplValues.ValueType.ImportName = ""
		tmplValues.ValueType.ImportPath = ""
	}
	if genPkg.PkgPath == tmplValues.KeyType.ImportPath {
		tmplValues.KeyType.ImportName = ""
		tmplValues.KeyType.ImportPath = ""
	}

	filename := strings.ToLower(tmplValues.Name) + "_gen.go"
	filepath := filepath.Join(wd, filename)

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, tmplValues); err != nil {
		return err
	}
	src, err := imports.Process(filepath, buf.Bytes(), nil)
	if err != nil {
		return err
	}
	if err := ioutil.WriteFile(filepath, src, 0644); err != nil {
		return err
	}
	return nil
}
