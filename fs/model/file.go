package model

import (
	"errors"
	"fmt"
	"github.com/google/cayley"
	"github.com/google/cayley/graph"
	"strconv"
)

const (
	parentLink = "hasParent"
	sizeLink   = "hasSize"
	nameLink   = "isNamed"
	attrLink   = "hasAttr"
)

type OFile struct {
	Id       string
	Name     string
	ParentId string
	Size     int64
	Attr     int64
}

func (of OFile) Iterator(graph *cayley.Handle) graph.Iterator {
	return cayley.StartPath(graph, of.Id).Out(parentLink).
		Or(cayley.StartPath(graph, of.Id).Out(nameLink)).
		Or(cayley.StartPath(graph, of.Id).Out(sizeLink)).
		Or(cayley.StartPath(graph, of.Id).Out(attrLink)).BuildIterator()
}

func (of OFile) Transaction() *graph.Transaction {
	transaction := cayley.NewTransaction()
	transaction.AddQuad(cayley.Quad(of.Id, parentLink, of.ParentId, ""))
	transaction.AddQuad(cayley.Quad(of.Id, nameLink, of.Name, ""))
	transaction.AddQuad(cayley.Quad(of.Id, sizeLink, fmt.Sprint(of.Size), ""))
	transaction.AddQuad(cayley.Quad(of.Id, attrLink, fmt.Sprint(of.Attr), ""))

	return transaction
}

func (of *OFile) SetProp(link, value string) (err error) {
	switch link {
	case parentLink:
		of.ParentId = value
	case nameLink:
		of.Name = value
	case sizeLink:
		of.Size, err = strconv.ParseInt(value, 10, 64)
	case attrLink:
		of.Attr, err = strconv.ParseInt(value, 10, 64)
	default:
		return errors.New(fmt.Sprint("No property for link name ", link))
	}

	return
}
