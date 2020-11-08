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
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(2)
	}
	err = generator.Generate(wd, *name, *keyType, *valueType)
	if err != nil {
		fmt.Fprintf(os.Stderr, err.Error())
		os.Exit(3)
	}
}
