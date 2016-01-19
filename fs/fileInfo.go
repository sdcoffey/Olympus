package fs

import "time"

type FileInfo struct {
	Id       string
	ParentId string
	Name     string
	Size     int64
	MTime    time.Time
	Attr     int64
}

type BlockInfo struct {
	Hash   string
	Offset int64
}

func FileInfoFromFile(file *OFile) FileInfo {
	var fileInfo FileInfo
	fileInfo.Id = file.Id
	fileInfo.Attr = int64(file.Mode())
	fileInfo.Name = file.Name()
	fileInfo.MTime = file.ModTime()
	fileInfo.ParentId = file.Parent().Id
	fileInfo.Size = file.Size()

	return fileInfo
}
