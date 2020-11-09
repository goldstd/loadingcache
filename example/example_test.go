package example_test

import (
	"testing"

	"github.com/Hartimer/loadingcache/example"
	"github.com/stretchr/testify/require"
)

func TestExample(t *testing.T) {
	e := example.New()
	e.Set("A", 10)
	e.Set("B", 5)
	e.Set("C", 2)

	require.Equal(t, int64(17), e.AddAges("A", "B", "C"))
}
