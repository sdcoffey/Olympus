package testutils

import (
	. "github.com/sdcoffey/olympus/Godeps/_workspace/src/gopkg.in/check.v1"
	"strings"
)

type isTrueChecker struct {
	*CheckerInfo
}

var IsTrue Checker = &isTrueChecker{
	&CheckerInfo{Name: "IsTrue", Params: []string{"value"}},
}

func (checker *isTrueChecker) Check(params []interface{}, names []string) (bool, string) {
	if param, ok := params[0].(bool); ok {
		return param, ""
	}
	return false, "Param passed not a boolean"
}

type containsChecker struct {
	*CheckerInfo
}

var Contains = &containsChecker{
	&CheckerInfo{Name: "Contains", Params: []string{"obtained", "substring"}},
}

func (checker *containsChecker) Check(params []interface{}, names []string) (bool, string) {
	if len(params) < 2 {
		return false, "Must pass two arguments to Contains Checker"
	}

	obtained, oko := params[0].(string)
	substring, okc := params[1].(string)
	if !oko || !okc {
		return false, "Both parameters must be strings"
	}

	return strings.Contains(obtained, substring), ""
}
