package graph

import "time"

type NodeInfo struct {
	Id       string
	ParentId string
	Name     string
	Size     int64
	MTime    time.Time
	Mode     uint32
}

type BlockInfo struct {
	Hash   string
	Offset int64
}
