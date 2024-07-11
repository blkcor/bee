package beeCache

import (
	"strings"
	"testing"
)

func TestSplitN(t *testing.T) {
	str := "a/b/c"
	s := strings.SplitN(str, "/", 2)
	t.Log(s)
}
