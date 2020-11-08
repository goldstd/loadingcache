package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Hartimer/loadingcache/cmd/typedcache/internal/generator"
)

func main() {
	var name = flag.String("name", "TypedCache", "Name of the typed cache type")
	var keyType = flag.String("key", "", "Type of the key")
	var valueType = flag.String("value", "", "Type of the value")
	var verbose = flag.Bool("verbose", false, "Enable verbose mode")
	flag.Parse()

	if keyType == nil || *keyType == "" {
		fmt.Fprintln(os.Stderr, "No key type specified")
		flag.Usage()
		os.Exit(1)
	}
	if valueType == nil || *valueType == "" {
		fmt.Fprintln(os.Stderr, "No value type specified")
		flag.Usage()
		os.Exit(1)
	}

	wd, err := os.Getwd()
	if err != nil {
		if verbose != nil && *verbose {
			fmt.Fprintf(os.Stderr, "%+v", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
		os.Exit(2)
	}
	err = generator.Generate(wd, *name, *keyType, *valueType)
	if err != nil {
		if verbose != nil && *verbose {
			fmt.Fprintf(os.Stderr, "%+v", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v", err)
		}
		os.Exit(3)
	}
}
