package api

import (
	"fmt"
	"regexp"
)

type Endpoint struct {
	template, query, Verb string
}

func newEndpoint(path, verb string) Endpoint {
	return Endpoint{
		template: path,
		Verb:     verb,
	}
}

func (e Endpoint) Build(args ...interface{}) Endpoint {
	var i int
	e.template = templateRegex.ReplaceAllStringFunc(e.template, func(s string) (value string) {
		if i < len(args) {
			value = fmt.Sprint(args[i])
		} else {
			value = ""
		}

		i++
		return
	})
	return e
}

func (e Endpoint) Query(key, val string) Endpoint {
	e.query += fmt.Sprint(key, "=", val)
	return e
}

func (e Endpoint) String() string {
	url := e.template
	if e.query != "" {
		url += "?" + e.query
	}
	return url
}

func (e Endpoint) Template() string {
	return e.template
}

var (
	ListNodes    = newEndpoint("/node/{parentId}", "GET")
	ListBlocks   = newEndpoint("/node/{nodeId}/block", "GET")
	CreateNode   = newEndpoint("/node/{parentId}", "POST")
	RemoveNode   = newEndpoint("/node/{nodeId}", "DELETE")
	UpdateNode   = newEndpoint("/node/{nodeId}", "PATCH")
	WriteBlock   = newEndpoint("/node/{nodeId}/block/{offset}", "PUT")
	ReadBlock    = newEndpoint("/node/{nodeId}/block/{offset}", "GET")
	DownloadNode = newEndpoint("/node/{nodeId}/stream", "GET")

	templateRegex = regexp.MustCompile("{(.*?)}")
)

type Encoding string

const (
	JsonEncoding Encoding = "application/json"
	XmlEncoding  Encoding = "application/xml"
	GobEncoding  Encoding = "application/gob"
)
