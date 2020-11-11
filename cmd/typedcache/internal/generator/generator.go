// Package generator logic is heavily inspired by https://github.com/vektah/dataloaden
package generator

import (
	"bytes"
	"html/template"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
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

// expectedPartsCount indicates how many parts we expect there to be after parsing a type string.
// E.g. []*github.com/import/path.Name
const expectedPartsCount = 4

var partsRe = regexp.MustCompile(`^([\[\]\*]*)(.*?)(\.\w*)?$`)

// parseType converts a string to a richer data structure.
// It is capable of parsing simple types, like "string" or
// imported types in the format github.com/import/path.Name.
func parseType(str string) (*goType, error) {
	parts := partsRe.FindStringSubmatch(str)
	if len(parts) != expectedPartsCount {
		return nil, errors.New("type must be in the form []*github.com/import/path.Name")
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
			return nil, errors.Wrap(err, "")
		}
		if len(p) != 1 {
			return nil, errors.New("not found")
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
		return nil, errors.Wrap(err, "")
	}

	if len(p) != 1 {
		return nil, errors.New("unable to find package info for " + dir)
	}

	return p[0], nil
}

// Generate generates the template and writes it to disk
func Generate(wd string, name string, keyType string, valueType string) error {
	tmplValues := templateValues{Name: name}

	var err error
	tmplValues.KeyType, err = parseType(keyType)
	if err != nil {
		return errors.Wrap(err, "faile to parse key type")
	}
	tmplValues.ValueType, err = parseType(valueType)
	if err != nil {
		return errors.Wrap(err, "failed to parse value type")
	}

	genPkg, err := getPackage(wd)
	if err != nil {
		return errors.Wrap(err, "failed to get package out of current working directory")
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
	tmpl := template.Must(template.New("typedCacheTemplate").Parse(typedCacheTemplate))
	if err := tmpl.Execute(&buf, tmplValues); err != nil {
		return errors.Wrap(err, "failed to execute template")
	}
	src, err := imports.Process(filepath, buf.Bytes(), nil)
	if err != nil {
		return errors.Wrap(err, "failed to apply go imports")
	}
	// Disabling gosec specifically here since we do want the generated code to be readable
	//nolint:gosec
	if err := ioutil.WriteFile(filepath, src, 0644); err != nil {
		return errors.Wrap(err, "failed to write file")
	}
	return nil
}
