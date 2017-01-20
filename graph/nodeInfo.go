package graph

import (
	"os"
	"time"
)

type NodeInfo struct {
	Id       string      `json:"id"`
	ParentId string      `json:"parent_id"`
	Name     string      `json:"name"`
	Size     int64       `json:"size"`
	MTime    time.Time   `json:"m_time"`
	Mode     os.FileMode `json:"mode"`
	Type     string      `json:"type"`
}

type BlockInfo struct {
	Hash   string `json:"hash"`
	Offset int64  `json:"offset"`
}
