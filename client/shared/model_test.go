package shared

import (
	"errors"
	"fmt"
	"io"
	"os"
	"testing"
	"time"

	"code.google.com/p/go-uuid/uuid"
	"github.com/google/cayley"
	"github.com/sdcoffey/olympus/fs"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	handle, _ := cayley.NewMemoryGraph()
	fs.Init(handle)
	exitCode := m.Run()
	handle.Close()
	os.Exit(exitCode)
}

func TestModel_returnsErrIfRootDoestNotExist(t *testing.T) {
	apiClient := fakeApiClient{}
	model := newModel(apiClient, "not-an-id")
	err := model.Init()
	assert.NotNil(t, err)

	assert.Equal(t, "Root with id: not-an-id does not exist", err.Error())
}

func TestModel_doesNotReturnErrorIfRootExists(t *testing.T) {
	apiClient := newFakeApiClient()
	model := newModel(apiClient, fs.RootNodeId)
	assert.Nil(t, model.Init())
}

func TestModel_InitRefreshesCache(t *testing.T) {
	assert.Nil(t, resetMemCache())
	apiClient := newFakeApiClient()
	model := newModel(apiClient, fs.RootNodeId)

	for i := 0; i < 3; i++ {
		info := fs.FileInfo{}
		info.Name = fmt.Sprint(i + 1)
		info.ParentId = fs.RootNodeId
		info.MTime = time.Now().UTC()
		info.Size = int64(i * 10)
		apiClient.AddFile(info)
	}

	err := model.Init()
	assert.Nil(t, err)

	assert.EqualValues(t, 3, len(model.Root.Children()))
}

func TestModel_refreshRemovesLocalItemsDeletedRemotely(t *testing.T) {
	assert.Nil(t, resetMemCache())
	apiClient := newFakeApiClient()
	model := newModel(apiClient, fs.RootNodeId)

	for i := 0; i < 3; i++ {
		info := fs.FileInfo{}
		info.Name = fmt.Sprint(i + 1)
		info.ParentId = fs.RootNodeId
		info.MTime = time.Now().UTC()
		info.Size = int64(i * 10)
		apiClient.AddFile(info)
	}

	err := model.Init()
	assert.Nil(t, err)

	assert.EqualValues(t, 3, len(model.Root.Children()))

	children := apiClient.fileMap[fs.RootNodeId]
	children = children[:2]
	apiClient.fileMap[fs.RootNodeId] = children

	err = model.Refresh()
	assert.Nil(t, err)
	assert.EqualValues(t, 2, len(model.Root.Children()))
}

func TestModel_findFileByNameReturnsCorrectFile(t *testing.T) {
	assert.Nil(t, resetMemCache())
	apiClient := newFakeApiClient()
	model := newModel(apiClient, fs.RootNodeId)

	for i := 0; i < 3; i++ {
		info := fs.FileInfo{}
		info.Name = fmt.Sprint(i + 1)
		info.ParentId = fs.RootNodeId
		info.MTime = time.Now().UTC()
		info.Size = int64(i * 10)
		apiClient.AddFile(info)
	}

	assert.NoError(t, model.Init())

	file := model.FindFileByName("1")
	assert.Equal(t, fs.RootNodeId, file.Parent().Id)
	assert.EqualValues(t, 0, file.Size())
}

func TestModel_absPathReturnsCorrectPathForPathWithUpDir(t *testing.T) {
	assert.Nil(t, resetMemCache())
	apiClient := newFakeApiClient()

	file1 := fs.FileInfo{
		Id:       "abcd",
		Name:     "A",
		ParentId: fs.RootNodeId,
		Attr:     int64(os.ModeDir),
	}
	file2 := fs.FileInfo{
		Id:       "efgh",
		Name:     "B",
		ParentId: "abcd",
		Attr:     0,
	}
	file3 := fs.FileInfo{
		Id:       "ijkl",
		Name:     "C",
		ParentId: "abcd",
		Attr:     0,
	}
	assert.NoError(t, apiClient.AddFile(file1))
	assert.NoError(t, apiClient.AddFile(file2))
	assert.NoError(t, apiClient.AddFile(file3))

	rootModel := newModel(apiClient, fs.RootNodeId)
	assert.NoError(t, rootModel.Init())

	model := newModel(apiClient, "efgh")
	assert.NoError(t, model.Init())

	path := "../C"
	abspath := model.absPath(path)

	assert.Equal(t, "/A/B/C", abspath)
}

// TODO: absPathReturnsSamePathForExistingAbsolutePath
// TODO: absPathReturnsRootWhenUpdirsExceedTreeDepth
// TODO: abspathReturnsFullPathForSingleLeaf

func resetMemCache() error {
	if handle, err := cayley.NewMemoryGraph(); err != nil {
		return err
	} else {
		return fs.Init(handle)
	}
}

type fakeApiClient struct {
	fileMap map[string][]fs.FileInfo
}

func newFakeApiClient() *fakeApiClient {
	client := new(fakeApiClient)
	client.fileMap = make(map[string][]fs.FileInfo)
	root := fs.FileInfo{}
	root.Id = fs.RootNodeId
	root.Attr = int64(os.ModeDir)
	root.Name = "root"

	client.fileMap[root.Id] = make([]fs.FileInfo, 0)
	return client
}

func (client *fakeApiClient) AddFile(file fs.FileInfo) error {
	fmt.Println(client.fileMap)
	if file.Id == "" {
		file.Id = uuid.New()
	}
	if children, ok := client.fileMap[file.ParentId]; ok {
		children = append(children, file)
		client.fileMap[file.ParentId] = children
		if file.Attr&int64(os.ModeDir) > 0 {
			client.fileMap[file.Id] = make([]fs.FileInfo, 0)
		}
		return nil
	} else {
		return errors.New("File with id: " + file.ParentId + " does not exist")
	}
}

func (client fakeApiClient) ListFiles(parentId string) ([]fs.FileInfo, error) {
	if files, ok := client.fileMap[parentId]; ok {
		return files, nil
	} else {
		return make([]fs.FileInfo, 0), errors.New("File with id: " + parentId + " does not exist")
	}
}

func (client fakeApiClient) CreateDirectory(parentId, name string) (string, error) {
	return "", nil
}

func (client fakeApiClient) MoveFile(fileid, newParentId, newName string) error {
	return nil
}

func (client fakeApiClient) RemoveFile(fileId string) error {
	return nil
}

func (client fakeApiClient) CreateFile(info fs.FileInfo) (fs.FileInfo, error) {
	return fs.FileInfo{}, nil
}

func (client fakeApiClient) UpdateFile(info fs.FileInfo) error {
	return nil
}

func (client fakeApiClient) HasBlocks(fileId string, blocks []string) ([]string, error) {
	return []string{}, nil
}

func (client fakeApiClient) SendBlock(fileId string, block fs.BlockInfo, data io.Reader) error {
	return nil
}
