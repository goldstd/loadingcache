package generator_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/Hartimer/loadingcache/cmd/typedcache/internal/generator"
	"github.com/stretchr/testify/require"
)

func TestGeneratorRegression(t *testing.T) {
	// We will use the current directory to the package makes sense
	wd, err := os.Getwd()
	require.NoError(t, err)

	// Using a temporary file so we can test in parallel if needed
	// The current test however does assume we know how the generator
	// picks file names.
	tmpfile, err := ioutil.TempFile(wd, "a*_gen.go")
	require.NoError(t, err)
	defer os.Remove(tmpfile.Name())

	// The name passed in to the generator informs the name of the file.
	// Again, this assumes understanding of the inner workings of the generator.
	filenameWithoutExtention := strings.TrimSuffix(filepath.Base(tmpfile.Name()), "_gen.go")

	cacheName := strings.Title(filenameWithoutExtention)
	require.NoError(t, generator.Generate(wd, cacheName, "string", "int64"))

	// Reading the expected output template.
	// We have to use a template since we're using temp files, which
	// do not have deterministic names.
	var buf bytes.Buffer
	require.NoError(t, template.Must(template.New("test").Parse(expectedOutput)).
		Execute(&buf, map[string]string{"Name": cacheName}))

	contents, err := ioutil.ReadAll(tmpfile)
	require.NoError(t, err)
	require.Equal(t, buf.String(), string(contents))
}
