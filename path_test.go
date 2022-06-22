package pathlib

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type PathSuite struct {
	suite.Suite
	tmpdir *Path
}

func (p *PathSuite) SetupTest() {
	// We actually can't use the MemMapFs because some of the tests
	// are testing symlink behavior. We might want to split these
	// tests out to use MemMapFs when possible.
	tmpdir, err := ioutil.TempDir("", "")
	require.NoError(p.T(), err)
	p.tmpdir = NewPath(tmpdir)
}

func (p *PathSuite) TeardownTest() {
	assert.NoError(p.T(), p.tmpdir.RemoveAll())
}

// -----------------------------------------------------------------------------
//
// test reimplemented pathlib.PurePath-like methods
//
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
//
// test XXX
//
// -----------------------------------------------------------------------------

func (p *PathSuite) TestSymlink() {
	symlink := p.tmpdir.Join("symlink")
	require.NoError(p.T(), symlink.Symlink(p.tmpdir))

	linkLocation, err := symlink.Readlink()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), p.tmpdir.String(), linkLocation.String())
}

func (p *PathSuite) TestSymlinkBadFs() {
	symlink := p.tmpdir.Join("symlink")
	symlink.fs = afero.NewMemMapFs()

	assert.Error(p.T(), symlink.Symlink(p.tmpdir))
}

func (p *PathSuite) TestJoin() {
	joined := p.tmpdir.Join("test1")
	assert.Equal(p.T(), filepath.Join(p.tmpdir.String(), "test1"), joined.String())
}

func (p *PathSuite) TestWriteAndRead() {
	expectedBytes := []byte("hello world!")
	file := p.tmpdir.Join("test.txt")
	require.NoError(p.T(), file.WriteFile(expectedBytes))
	bytes, err := file.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), expectedBytes, bytes)
}

func (p *PathSuite) TestChmod() {
	file := p.tmpdir.Join("file1.txt")
	require.NoError(p.T(), file.WriteFile([]byte("")))

	file.Chmod(0o777)
	fileInfo, err := file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o777), fileInfo.Mode()&os.ModePerm)

	file.Chmod(0o755)
	fileInfo, err = file.Stat()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), os.FileMode(0o755), fileInfo.Mode()&os.ModePerm)
}

func (p *PathSuite) TestMkdir() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.Mkdir())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirParentsDontExist() {
	subdir := p.tmpdir.Join("subdir1", "subdir2")
	assert.Error(p.T(), subdir.Mkdir())
}

func (p *PathSuite) TestMkdirAll() {
	subdir := p.tmpdir.Join("subdir")
	assert.NoError(p.T(), subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestMkdirAllMultipleSubdirs() {
	subdir := p.tmpdir.Join("subdir1", "subdir2", "subdir3")
	assert.NoError(p.T(), subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(p.T(), err)
	assert.True(p.T(), isDir)
}

func (p *PathSuite) TestRenameString() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte("hello world!")))

	newPath := p.tmpdir.Join("file2.txt")

	renamedPath, err := file.RenamePath(newPath)
	assert.NoError(p.T(), err)
	assert.Equal(p.T(), renamedPath.String(), p.tmpdir.Join("file2.txt").String())

	newBytes, err := renamedPath.ReadFile()
	require.NoError(p.T(), err)
	assert.Equal(p.T(), []byte("hello world!"), newBytes)

	renamedFileExists, err := renamedPath.Exists()
	require.NoError(p.T(), err)
	assert.True(p.T(), renamedFileExists)

	oldFileExists, err := p.tmpdir.Join("file.txt").Exists()
	require.NoError(p.T(), err)
	assert.False(p.T(), oldFileExists)
}

func (p *PathSuite) TestSizeZero() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte{}))
	size, err := file.Size()
	require.NoError(p.T(), err)
	p.Zero(size)
}

func (p *PathSuite) TestSizeNonZero() {
	msg := "oh, it's you"
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte(msg)))
	size, err := file.Size()
	require.NoError(p.T(), err)
	p.Equal(len(msg), int(size))
}

func (p *PathSuite) TestIsDir() {
	dir := p.tmpdir.Join("dir")
	require.NoError(p.T(), dir.Mkdir())
	isDir, err := dir.IsDir()
	require.NoError(p.T(), err)
	p.True(isDir)
}

func (p *PathSuite) TestIsntDir() {
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte("hello world!")))
	isDir, err := file.IsDir()
	require.NoError(p.T(), err)
	p.False(isDir)
}

func (p *PathSuite) TestGetLatest() {
	now := time.Now()
	for i := 0; i < 5; i++ {
		file := p.tmpdir.Join(fmt.Sprintf("file%d.txt", i))
		require.NoError(p.T(), file.WriteFile([]byte(fmt.Sprintf("hello %d", i))))
		require.NoError(p.T(), file.Chtimes(now, now))
		now = now.Add(time.Duration(1) * time.Hour)
	}

	latest, err := p.tmpdir.GetLatest()
	require.NoError(p.T(), err)

	assert.Equal(p.T(), "file4.txt", latest.Name())
}

func (p *PathSuite) TestGetLatestEmpty() {
	latest, err := p.tmpdir.GetLatest()
	require.NoError(p.T(), err)
	assert.Nil(p.T(), latest)
}

func (p *PathSuite) TestOpen() {
	msg := "cubs > cardinals"
	file := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file.WriteFile([]byte(msg)))
	fileHandle, err := file.Open()
	require.NoError(p.T(), err)

	readBytes := make([]byte, len(msg)+5)
	n, err := fileHandle.Read(readBytes)
	p.Equal(len(msg), n)
	p.Equal(msg, string(readBytes[0:n]))
}

func (p *PathSuite) TestOpenFile() {
	file := p.tmpdir.Join("file.txt")
	fileHandle, err := file.OpenFile(os.O_RDWR | os.O_CREATE)
	require.NoError(p.T(), err)

	msg := "do you play croquet?"
	n, err := fileHandle.WriteString(msg)
	p.Equal(len(msg), n)
	p.NoError(err)

	bytes := make([]byte, len(msg)+5)
	n, err = fileHandle.ReadAt(bytes, 0)
	p.Equal(len(msg), n)
	p.True(errors.Is(err, io.EOF))
	p.Equal(msg, string(bytes[0:n]))
}

func (p *PathSuite) TestDirExists() {
	dir1 := p.tmpdir.Join("subdir")
	exists, err := dir1.DirExists()
	require.NoError(p.T(), err)
	p.False(exists)

	require.NoError(p.T(), dir1.Mkdir())
	exists, err = dir1.DirExists()
	require.NoError(p.T(), err)
	p.True(exists)
}

func (p *PathSuite) TestIsFile() {
	file1 := p.tmpdir.Join("file.txt")

	require.NoError(p.T(), file1.WriteFile([]byte("")))
	exists, err := file1.IsFile()
	require.NoError(p.T(), err)
	p.True(exists)
}

func (p *PathSuite) TestIsEmpty() {
	file1 := p.tmpdir.Join("file.txt")

	require.NoError(p.T(), file1.WriteFile([]byte("")))
	isEmpty, err := file1.IsEmpty()
	require.NoError(p.T(), err)
	p.True(isEmpty)
}

func (p *PathSuite) TestIsSymlink() {
	file1 := p.tmpdir.Join("file.txt")
	require.NoError(p.T(), file1.WriteFile([]byte("")))

	symlink := p.tmpdir.Join("symlink")
	p.NoError(symlink.Symlink(file1))
	isSymlink, err := symlink.IsSymlink()
	require.NoError(p.T(), err)
	p.True(isSymlink)

	stat, _ := symlink.Stat()
	p.T().Logf("%v", stat.Mode())
	p.T().Logf(symlink.String())
}

func (p *PathSuite) TestResolveAll() {
	home := p.tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp")
	require.NoError(p.T(), home.MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(NewPath("../data/users/home")))
	require.NoError(p.T(), p.tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := p.tmpdir.Join("home/LandonTClipp").ResolveAll()
	p.T().Log(resolved.String())
	require.NoError(p.T(), err)

	homeResolved, err := home.ResolveAll()
	require.NoError(p.T(), err)

	p.Equal(homeResolved.Clean().String(), resolved.Clean().String())
}

func (p *PathSuite) TestResolveAllAbsolute() {
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(p.T(), p.tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(p.tmpdir.Join("mnt", "nfs", "data", "users", "home")))
	require.NoError(p.T(), p.tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := p.tmpdir.Join("home", "LandonTClipp").ResolveAll()
	p.NoError(err)
	resolvedParts := resolved.Parts()
	p.Equal(
		strings.Join(
			[]string{"mnt", "nfs", "data", "users", "home", "LandonTClipp"}, resolved.flavor.Separator(),
		),
		strings.Join(resolvedParts[len(resolvedParts)-6:], resolved.flavor.Separator()))
}

func (p *PathSuite) TestEquals() {
	hello1 := p.tmpdir.Join("hello", "world")
	require.NoError(p.T(), hello1.MkdirAll())
	hello2 := p.tmpdir.Join("hello", "world")
	require.NoError(p.T(), hello2.MkdirAll())

	p.True(hello1.Equals(hello2))
}

func (p *PathSuite) TestDeepEquals() {
	hello := p.tmpdir.Join("hello.txt")
	require.NoError(p.T(), hello.WriteFile([]byte("hello")))
	symlink := p.tmpdir.Join("symlink")
	require.NoError(p.T(), symlink.Symlink(hello))

	equals, err := hello.DeepEquals(symlink)
	p.NoError(err)
	p.True(equals)
}

func (p *PathSuite) TestReadDir() {
	require.NoError(p.T(), TwoFilesAtRootTwoInSubdir(p.tmpdir))
	paths, err := p.tmpdir.ReadDir()
	p.NoError(err)
	p.Equal(3, len(paths))
}

func (p *PathSuite) TestReadDirInvalidString() {
	paths, err := p.tmpdir.Join("i_dont_exist").ReadDir()
	p.Error(err)
	p.Equal(0, len(paths))
}

func (p *PathSuite) TestCreate() {
	msg := "hello world"
	file, err := p.tmpdir.Join("hello.txt").Create()
	p.NoError(err)
	defer file.Close()
	n, err := file.WriteString(msg)
	p.Equal(len(msg), n)
	p.NoError(err)
}

func (p *PathSuite) TestGlobFunction() {
	hello1 := p.tmpdir.Join("hello1.txt")
	require.NoError(p.T(), hello1.WriteFile([]byte("hello")))

	hello2 := p.tmpdir.Join("hello2.txt")
	require.NoError(p.T(), hello2.WriteFile([]byte("hello2")))

	paths, err := p.tmpdir.Glob("hello1*")
	p.NoError(err)
	require.Equal(p.T(), 1, len(paths))
	p.True(hello1.Equals(paths[0]), "received an unexpected path: %v", paths[0])
}

func TestPathSuite(t *testing.T) {
	suite.Run(t, new(PathSuite))
}
