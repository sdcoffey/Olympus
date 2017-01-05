package checkers

import (
	. "gopkg.in/check.v1"
	"strings"
	"time"
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

type equalTimeChecker struct {
	*CheckerInfo
}

var EqualTime = &equalTimeChecker{
	&CheckerInfo{Name: "EqualTime", Params: []string{"time 1", "time 2", "tolerance(opt)"}},
}

func (checker *equalTimeChecker) Check(params []interface{}, names []string) (bool, string) {
	if len(params) < 2 {
		return false, "Must pass two arguments to Equal Time Checker"
	}

	time1, ok1 := params[0].(time.Time)
	time2, ok2 := params[1].(time.Time)
	if !ok1 || !ok2 {
		return false, "Both parameters must be time.Time instances"
	}

	d := time.Duration(0)
	if len(params) == 3 {
		d = params[2].(time.Duration)
	}

	return time1.Sub(time2) <= d, ""
}

type withinNowChecker struct {
	*CheckerInfo
}

var WithinNow = &withinNowChecker{
	&CheckerInfo{Name: "WithinNow", Params: []string{"time", "tolerance"}},
}

func (checker *withinNowChecker) Check(params []interface{}, names []string) (bool, string) {
	if len(params) < 2 {
		return false, "Must pass two arguments to Equal Time Checker"
	}

	var t time.Time
	var d time.Duration
	var ok bool

	if t, ok = params[0].(time.Time); !ok {
		return false, "First parameter must be a time.Time struct"
	} else if d, ok = params[1].(time.Duration); !ok {
		return false, "Second parameter must be a time.Duration instance"
	}

	return time.Now().Sub(t) <= d, ""
}
