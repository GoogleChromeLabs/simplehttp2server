package main

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	parensExpr   = regexp.MustCompile("([^\\\\])\\(([^)]+)\\)")
	questionExpr = regexp.MustCompile("\\?([^(])")
)

func CompileExtGlob(extglob string) (*regexp.Regexp, error) {
	tmp := extglob
	tmp = strings.Replace(tmp, ".", "\\.", -1)
	tmp = strings.Replace(tmp, "**", ".🐷", -1)
	tmp = strings.Replace(tmp, "*", fmt.Sprintf("[^%c]*", os.PathSeparator), -1)
	tmp = questionExpr.ReplaceAllString(tmp, fmt.Sprintf("[^%c]$1", os.PathSeparator))
	tmp = parensExpr.ReplaceAllString(tmp, "($2)$1")
	tmp = strings.Replace(tmp, ")@", ")", -1)
	tmp = strings.Replace(tmp, ".🐷", ".*", -1)
	return regexp.Compile("^" + tmp + "$")
}
