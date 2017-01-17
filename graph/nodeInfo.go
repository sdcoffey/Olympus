package graph

import (
	"os"
	"time"
)

type NodeInfo struct {
	Id       string
	ParentId string
	Name     string
	Size     int64
	MTime    time.Time
	Mode     os.FileMode
	Type     string
}

type BlockInfo struct {
	Hash   string
	Offset int64
	Size 	int64
}
