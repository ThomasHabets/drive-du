package lib

/*
 * This file contains pretty-printing functions.
 */

import (
	"fmt"
	"regexp"
)

var (
	prettyRE = regexp.MustCompile(`(\d)(\d{3})($|,)`)
)

func Pretty(n int64) string {
	ret := fmt.Sprint(n)
	for {
		n := prettyRE.ReplaceAllString(ret, "$1,$2$3")
		if n == ret {
			return n
		}
		ret = n
	}
}
