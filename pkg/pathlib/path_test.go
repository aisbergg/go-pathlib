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

	"github.com/aisbergg/go-pathlib/internal/testutils"
	"github.com/spf13/afero"
)

func setupPathTest(t *testing.T) (testutils.Assertions, testutils.Assertions, Path) {
	assert := testutils.NewAssert(t)
	require := testutils.NewRequire(t)

	// We actually can't use the MemMapFs because some of the tests
	// are testing symlink behavior. We might want to split these
	// tests out to use MemMapFs when possible.
	tmpdir, err := ioutil.TempDir("", "")
	testutils.NewRequire(t).NoError(err)
	return assert, require, NewPath(tmpdir)
}

func teardownPathTest(t *testing.T, tmpdir Path) {
	testutils.NewAssert(t).NoError(tmpdir.RemoveAll())
}

func TestSymlink(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	symlink := tmpdir.Join("symlink")
	require.NoError(symlink.Symlink(tmpdir))

	linkLocation, err := symlink.Readlink()
	require.NoError(err)
	assert.Equal(tmpdir.String(), linkLocation.String())
}

func TestSymlinkBadFs(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	symlink := tmpdir.Join("symlink")
	symlink.fs = afero.NewMemMapFs()

	assert.Error(symlink.Symlink(tmpdir))
}

func TestJoin(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	joined := tmpdir.Join("test1")
	assert.Equal(filepath.Join(tmpdir.String(), "test1"), joined.String())
}

func TestWriteAndRead(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	expectedBytes := []byte("hello world!")
	file := tmpdir.Join("test.txt")
	require.NoError(file.WriteFile(expectedBytes))
	bytes, err := file.ReadFile()
	require.NoError(err)
	assert.Equal(expectedBytes, bytes)
}

func TestChmod(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file := tmpdir.Join("file1.txt")
	require.NoError(file.WriteFile([]byte("")))

	file.Chmod(0o777)
	fileInfo, err := file.Stat()
	require.NoError(err)

	assert.Equal(os.FileMode(0o777), fileInfo.Mode()&os.ModePerm)

	file.Chmod(0o755)
	fileInfo, err = file.Stat()
	require.NoError(err)

	assert.Equal(os.FileMode(0o755), fileInfo.Mode()&os.ModePerm)
}

func TestMkdir(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	subdir := tmpdir.Join("subdir")
	assert.NoError(subdir.Mkdir())
	isDir, err := subdir.IsDir()
	require.NoError(err)
	assert.True(isDir)
}

func TestMkdirParentsDontExist(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	subdir := tmpdir.Join("subdir1", "subdir2")
	assert.Error(subdir.Mkdir())
}

func TestMkdirAll(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	subdir := tmpdir.Join("subdir")
	assert.NoError(subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(err)
	assert.True(isDir)
}

func TestMkdirAllMultipleSubdirs(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	subdir := tmpdir.Join("subdir1", "subdir2", "subdir3")
	assert.NoError(subdir.MkdirAll())
	isDir, err := subdir.IsDir()
	require.NoError(err)
	assert.True(isDir)
}

func TestRenameString(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file := tmpdir.Join("file.txt")
	require.NoError(file.WriteFile([]byte("hello world!")))

	newPath := tmpdir.Join("file2.txt")

	renamedPath, err := file.RenamePath(newPath)
	assert.NoError(err)
	assert.Equal(renamedPath.String(), tmpdir.Join("file2.txt").String())

	newBytes, err := renamedPath.ReadFile()
	require.NoError(err)
	assert.Equal([]byte("hello world!"), newBytes)

	renamedFileExists, err := renamedPath.Exists()
	require.NoError(err)
	assert.True(renamedFileExists)

	oldFileExists, err := tmpdir.Join("file.txt").Exists()
	require.NoError(err)
	assert.False(oldFileExists)
}

func TestSizeZero(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file := tmpdir.Join("file.txt")
	require.NoError(file.WriteFile([]byte{}))
	size, err := file.Size()
	require.NoError(err)
	assert.Equal(size, int64(0))
}

func TestSizeNonZero(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	msg := "oh, it's you"
	file := tmpdir.Join("file.txt")
	require.NoError(file.WriteFile([]byte(msg)))
	size, err := file.Size()
	require.NoError(err)
	assert.Equal(len(msg), int(size))
}

func TestIsDir(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	dir := tmpdir.Join("dir")
	require.NoError(dir.Mkdir())
	isDir, err := dir.IsDir()
	require.NoError(err)
	assert.True(isDir)
}

func TestIsntDir(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file := tmpdir.Join("file.txt")
	require.NoError(file.WriteFile([]byte("hello world!")))
	isDir, err := file.IsDir()
	require.NoError(err)
	assert.False(isDir)
}

func TestGetLatest(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	now := time.Now()
	for i := 0; i < 5; i++ {
		file := tmpdir.Join(fmt.Sprintf("file%d.txt", i))
		require.NoError(file.WriteFile([]byte(fmt.Sprintf("hello %d", i))))
		require.NoError(file.Chtimes(now, now))
		now = now.Add(time.Duration(1) * time.Hour)
	}

	latest, err := tmpdir.GetLatest()
	require.NoError(err)

	assert.Equal("file4.txt", latest.Name())
}

func TestGetLatestEmpty(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	_, err := tmpdir.GetLatest()
	assert.EqualError(ErrDirectoryEmpty, err)
}

func TestOpen(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	msg := "cubs > cardinals"
	file := tmpdir.Join("file.txt")
	require.NoError(file.WriteFile([]byte(msg)))
	fileHandle, err := file.Open()
	require.NoError(err)

	readBytes := make([]byte, len(msg)+5)
	n, err := fileHandle.Read(readBytes)
	assert.Equal(len(msg), n)
	assert.Equal(msg, string(readBytes[0:n]))
}

func TestOpenFile(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file := tmpdir.Join("file.txt")
	fileHandle, err := file.OpenFile(os.O_RDWR | os.O_CREATE)
	require.NoError(err)

	msg := "do you play croquet?"
	n, err := fileHandle.WriteString(msg)
	assert.Equal(len(msg), n)
	assert.NoError(err)

	bytes := make([]byte, len(msg)+5)
	n, err = fileHandle.ReadAt(bytes, 0)
	assert.Equal(len(msg), n)
	assert.True(errors.Is(err, io.EOF))
	assert.Equal(msg, string(bytes[0:n]))
}

func TestDirExists(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	dir1 := tmpdir.Join("subdir")
	exists, err := dir1.DirExists()
	require.NoError(err)
	assert.False(exists)

	require.NoError(dir1.Mkdir())
	exists, err = dir1.DirExists()
	require.NoError(err)
	assert.True(exists)
}

func TestIsFile(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file1 := tmpdir.Join("file.txt")

	require.NoError(file1.WriteFile([]byte("")))
	exists, err := file1.IsFile()
	require.NoError(err)
	assert.True(exists)
}

func TestIsEmpty(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file1 := tmpdir.Join("file.txt")

	require.NoError(file1.WriteFile([]byte("")))
	isEmpty, err := file1.IsEmpty()
	require.NoError(err)
	assert.True(isEmpty)
}

func TestIsSymlink(t *testing.T) {
	require := testutils.NewRequire(t)
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	file1 := tmpdir.Join("file.txt")
	require.NoError(file1.WriteFile([]byte("")))

	symlink := tmpdir.Join("symlink")
	assert.NoError(symlink.Symlink(file1))
	isSymlink, err := symlink.IsSymlink()
	require.NoError(err)
	assert.True(isSymlink)

	stat, _ := symlink.Stat()
	t.Logf("%v", stat.Mode())
	t.Logf(symlink.String())
}

func TestResolveAll(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	home := tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp")
	require.NoError(home.MkdirAll())
	require.NoError(tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(NewPath("../data/users/home")))
	require.NoError(tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := tmpdir.Join("home/LandonTClipp").ResolveAll()
	t.Log(resolved.String())
	require.NoError(err)

	homeResolved, err := home.ResolveAll()
	require.NoError(err)

	assert.Equal(homeResolved.Clean().String(), resolved.Clean().String())
}

func TestResolveAllAbsolute(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	require.NoError(tmpdir.Join("mnt", "nfs", "data", "users", "home", "LandonTClipp").MkdirAll())
	require.NoError(tmpdir.Join("mnt", "nfs", "symlinks").MkdirAll())
	require.NoError(tmpdir.Join("mnt", "nfs", "symlinks", "home").Symlink(tmpdir.Join("mnt", "nfs", "data", "users", "home")))
	require.NoError(tmpdir.Join("home").Symlink(NewPath("./mnt/nfs/symlinks/home")))

	resolved, err := tmpdir.Join("home", "LandonTClipp").ResolveAll()
	assert.NoError(err)
	resolvedParts := resolved.Parts()
	assert.Equal(
		strings.Join(
			[]string{"mnt", "nfs", "data", "users", "home", "LandonTClipp"}, resolved.flavor.Separator(),
		),
		strings.Join(resolvedParts[len(resolvedParts)-6:], resolved.flavor.Separator()))
}

func TestEquals(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	hello1 := tmpdir.Join("hello", "world")
	require.NoError(hello1.MkdirAll())
	hello2 := tmpdir.Join("hello", "world")
	require.NoError(hello2.MkdirAll())

	assert.True(hello1.Equals(hello2))
}

func TestDeepEquals(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	hello := tmpdir.Join("hello.txt")
	require.NoError(hello.WriteFile([]byte("hello")))
	symlink := tmpdir.Join("symlink")
	require.NoError(symlink.Symlink(hello))

	equals, err := hello.DeepEquals(symlink)
	assert.NoError(err)
	assert.True(equals)
}

func TestReadDir(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	require.NoError(TwoFilesAtRootTwoInSubdir(tmpdir))
	paths, err := tmpdir.ReadDir()
	assert.NoError(err)
	assert.Equal(3, len(paths))
}

func TestReadDirInvalidString(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	paths, err := tmpdir.Join("i_dont_exist").ReadDir()
	assert.Error(err)
	assert.Equal(0, len(paths))
}

func TestCreate(t *testing.T) {
	assert, _, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	msg := "hello world"
	file, err := tmpdir.Join("hello.txt").Create()
	assert.NoError(err)
	defer file.Close()
	n, err := file.WriteString(msg)
	assert.Equal(len(msg), n)
	assert.NoError(err)
}

func TestGlobFunction(t *testing.T) {
	assert, require, tmpdir := setupPathTest(t)
	defer teardownPathTest(t, tmpdir)
	hello1 := tmpdir.Join("hello1.txt")
	require.NoError(hello1.WriteFile([]byte("hello")))

	hello2 := tmpdir.Join("hello2.txt")
	require.NoError(hello2.WriteFile([]byte("hello2")))

	paths, err := tmpdir.Glob("hello1*")
	assert.NoError(err)
	require.Equal(1, len(paths))
	assert.True(hello1.Equals(paths[0]), "received an unexpected path: %v", paths[0])
}

// -----------------------------------------------------------------------------
//
// Benchmarks
//
// -----------------------------------------------------------------------------

func BenchmarkPath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i := 0; i < 200; i++ {
			p := NewPath("/a/b/c/d/e/f/g/h/i/j")
			_ = p.Parent().Parent().Parent().Parent().Parent()
			p2 := NewPath("/a/b/c/d/e/f")
			p, err := p.RelativeToPath(p2)
			if err != nil {
				b.Error(err)
			}
			_, err = p.WithName("foo")
			_, err = p.WithSuffix(".ooo")
			p.Clean()
		}
	}
}
