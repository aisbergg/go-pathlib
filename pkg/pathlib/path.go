package pathlib

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/spf13/afero"
)

// Path represents a filesystem path, offers methods for manipulating it and
// allows I/O operations by utilizing Afero.
type Path struct {
	PurePath
	fs afero.Fs

	// DefaultFileMode is the mode that is used when creating new files in
	// functions that do not accept os.FileMode as a parameter.
	DefaultFileMode os.FileMode
	// DefaultDirMode is the mode that will be used when creating new
	// directories.
	DefaultDirMode os.FileMode
}

// NewPath returns a new `Path` from the given path(s). Depending on the OS
// either a Windows or Posix flavored path is created.
func NewPath(paths ...string) Path {
	return NewPathWithFS(afero.NewOsFs(), paths...)
}

// NewPathWithFS returns a new `Path` from the given path(s). Depending on the OS
// either a Windows or Posix flavored path is created.
func NewPathWithFS(fs afero.Fs, paths ...string) Path {
	if runtime.GOOS == "windows" {
		return newPathWithFlavor(newWindowsFlavor(), fs, paths...)
	}
	return newPathWithFlavor(newPosixFlavor(), fs, paths...)
}

// NewPosixPath returns a new Posix flavored `Path` from the given
// path(s).
func NewPosixPath(paths ...string) Path {
	return NewPosixPathWithFS(afero.NewOsFs(), paths...)
}

// NewPosixPathWithFS returns a new Posix flavored `Path` from the given
// path(s).
func NewPosixPathWithFS(fs afero.Fs, paths ...string) Path {
	return newPathWithFlavor(newPosixFlavor(), fs, paths...)
}

// NewWindowsPath returns a new Windows flavored `Path` from the given
// path(s).
func NewWindowsPath(paths ...string) Path {
	return NewWindowsPathWithFS(afero.NewOsFs(), paths...)
}

// NewWindowsPathWithFS returns a new Windows flavored `Path` from the given
// path(s).
func NewWindowsPathWithFS(fs afero.Fs, paths ...string) Path {
	return newPathWithFlavor(newWindowsFlavor(), fs, paths...)
}

// newPathWithFlavor returns a new `Path` from the given path(s) and flavor.
func newPathWithFlavor(flavor flavorer, fs afero.Fs, paths ...string) Path {
	drive, root, parts := parseParts(paths, flavor)
	return Path{
		PurePath: PurePath{
			drive:  drive,
			root:   root,
			parts:  parts,
			flavor: flavor,
		},
		fs:              fs,
		DefaultFileMode: DefaultFileMode,
		DefaultDirMode:  DefaultDirMode,
	}
}

// copyPathWithPaths returns a copy with new path(s).
func copyPathWithPaths(copyFrom Path, paths ...string) Path {
	drive, root, parts := parseParts(paths, copyFrom.flavor)
	return Path{
		PurePath: PurePath{
			drive:  drive,
			root:   root,
			parts:  parts,
			flavor: copyFrom.flavor,
		},
		fs:              copyFrom.fs,
		DefaultFileMode: copyFrom.DefaultFileMode,
		DefaultDirMode:  copyFrom.DefaultDirMode,
	}
}

// copyPathWithPurePath returns a copy with a new underlying PurePath.
func copyPathWithPurePath(copyFrom Path, purePath PurePath) Path {
	return Path{
		PurePath:        purePath,
		fs:              copyFrom.fs,
		DefaultFileMode: copyFrom.DefaultFileMode,
		DefaultDirMode:  copyFrom.DefaultDirMode,
	}
}

type namer interface {
	Name() string
}

func getFsName(fs afero.Fs) string {
	if name, ok := fs.(namer); ok {
		return name.Name()
	}
	return ""
}

// Fs returns the internal afero.Fs object.
func (p Path) Fs() afero.Fs {
	return p.fs
}

func (p Path) doesNotImplementErr(interfaceName string) error {
	return doesNotImplementErr(interfaceName, p.Fs())
}

func doesNotImplementErr(interfaceName string, fs afero.Fs) error {
	return fmt.Errorf("%w: Path's afero filesystem %s does not implement %s", ErrDoesNotImplement, getFsName(fs), interfaceName)
}

func (p Path) lstatNotPossible() error {
	return lstatNotPossible(p.Fs())
}

func lstatNotPossible(fs afero.Fs) error {
	return fmt.Errorf("%w: Path's afero filesystem %s does not support lstat", ErrLstatNotPossible, getFsName(fs))
}

// -----------------------------------------------------------------------------
//
// afero.Fs wrappers
//
// -----------------------------------------------------------------------------

// Create creates a file if possible, returning the file and an error, if any happens.
func (p Path) Create() (File, error) {
	file, err := p.Fs().Create(p.String())
	return File{file}, err
}

// Mkdir makes the current dir. If the parents don't exist, an error
// is returned.
func (p Path) Mkdir(perm ...os.FileMode) error {
	mode := p.DefaultDirMode
	if len(perm) > 0 {
		mode = perm[0]
	}
	return p.Fs().Mkdir(p.String(), mode)
}

// MkdirAll makes all of the directories up to, and including, the given path.
func (p Path) MkdirAll(perm ...os.FileMode) error {
	mode := p.DefaultDirMode
	if len(perm) > 0 {
		mode = perm[0]
	}
	return p.Fs().MkdirAll(p.String(), mode)
}

// Open opens a file for read-only, returning it or an error, if any happens.
func (p Path) Open() (*File, error) {
	handle, err := p.Fs().Open(p.String())
	return &File{
		File: handle,
	}, err
}

// OpenFile opens a file using the given flags and (optionally) given mode.
// See the list of flags at: https://golang.org/pkg/os/#pkg-constants
func (p Path) OpenFile(flag int, perm ...os.FileMode) (*File, error) {
	mode := p.DefaultFileMode
	if len(perm) > 0 {
		mode = perm[0]
	}
	handle, err := p.Fs().OpenFile(p.String(), flag, mode)
	return &File{
		File: handle,
	}, err
}

// Remove removes a file, returning an error, if any
// happens.
func (p Path) Remove() error {
	return p.Fs().Remove(p.String())
}

// RemoveAll removes the given path and all of its children.
func (p Path) RemoveAll() error {
	return p.Fs().RemoveAll(p.String())
}

// Rename renames the path to the given target path.
func (p Path) Rename(target string) (Path, error) {
	newPath := copyPathWithPaths(p, target)
	if err := p.Fs().Rename(p.String(), newPath.String()); err != nil {
		return Path{}, err
	}
	return newPath, nil
}

// RenamePath renames the path to the given target path.
func (p Path) RenamePath(target Path) (Path, error) {
	return p.Rename(target.String())
}

// Stat returns the os.FileInfo of the path.
func (p Path) Stat() (os.FileInfo, error) {
	return p.Fs().Stat(p.String())
}

// Chmod changes the file mode of the given path
func (p Path) Chmod(mode os.FileMode) error {
	return p.Fs().Chmod(p.String(), mode)
}

// Chtimes changes the modification and access time of the given path.
func (p Path) Chtimes(atime time.Time, mtime time.Time) error {
	return p.Fs().Chtimes(p.String(), atime, mtime)
}

// -----------------------------------------------------------------------------
//
// afero.Afero wrappers
//
// -----------------------------------------------------------------------------

// DirExists returns whether or not the path represents a directory that exists
func (p Path) DirExists() (bool, error) {
	return afero.DirExists(p.Fs(), p.String())
}

// Exists returns whether the path exists
func (p Path) Exists() (bool, error) {
	return afero.Exists(p.Fs(), p.String())
}

// FileContainsAnyBytes returns whether or not the path contains
// any of the listed bytes.
func (p Path) FileContainsAnyBytes(subslices [][]byte) (bool, error) {
	return afero.FileContainsAnyBytes(p.Fs(), p.String(), subslices)
}

// FileContainsBytes returns whether or not the given file contains the bytes
func (p Path) FileContainsBytes(subslice []byte) (bool, error) {
	return afero.FileContainsBytes(p.Fs(), p.String(), subslice)
}

// IsDir checks if a given path is a directory.
func (p Path) IsDir() (bool, error) {
	return afero.IsDir(p.Fs(), p.String())
}

// IsDir returns whether or not the os.FileMode object represents a
// directory.
func IsDir(mode os.FileMode) bool {
	return mode.IsDir()
}

// IsEmpty checks if a given file or directory is empty.
func (p Path) IsEmpty() (bool, error) {
	return afero.IsEmpty(p.Fs(), p.String())
}

// ReadDir reads the current path and returns a list of the corresponding
// Path objects. This function differs from os.Readdir in that it does
// not call Stat() on the files. Instead, it calls Readdirnames which
// is less expensive and does not force the caller to make expensive
// Stat calls.
func (p Path) ReadDir() ([]Path, error) {
	var paths []Path
	handle, err := p.Open()
	if err != nil {
		return paths, err
	}
	children, err := handle.Readdirnames(-1)
	if err != nil {
		return paths, err
	}
	for _, child := range children {
		paths = append(paths, p.Join(child))
	}
	return paths, err
}

// ReadFile reads the given path and returns the data. If the file doesn't exist
// or is a directory, an error is returned.
func (p Path) ReadFile() ([]byte, error) {
	return afero.ReadFile(p.Fs(), p.String())
}

// SafeWriteReader is the same as WriteReader but checks to see if file/directory already exists.
func (p Path) SafeWriteReader(r io.Reader) error {
	return afero.SafeWriteReader(p.Fs(), p.String(), r)
}

// WriteFile writes the given data to the path (if possible). If the file exists,
// the file is truncated. If the file is a directory, or the path doesn't exist,
// an error is returned.
func (p Path) WriteFile(data []byte, perm ...os.FileMode) error {
	mode := p.DefaultFileMode
	if len(perm) > 0 {
		mode = perm[0]
	}
	return afero.WriteFile(p.Fs(), p.String(), data, mode)
}

// WriteReader takes a reader and writes the content
func (p Path) WriteReader(r io.Reader) error {
	return afero.WriteReader(p.Fs(), p.String(), r)
}

// -----------------------------------------------------------------------------
//
// reimplemented pathlib.PurePath-like methods
//
// -----------------------------------------------------------------------------

// WithName returns a new path with the file name changed.
func (p Path) WithName(name string) (Path, error) {
	pp, err := p.PurePath.WithName(name)
	if err != nil {
		return Path{}, err
	}
	return copyPathWithPurePath(p, pp), nil
}

// WithStem returns a new path with the stem changed.
func (p Path) WithStem(stem string) (Path, error) {
	return p.WithName(stem + p.Suffix())
}

// WithSuffix returns a new path with the file suffix changed. If the path has
// no suffix, the suffix is added; if the suffix is empty, the suffix is removed
// from the path.
func (p Path) WithSuffix(suffix string) (Path, error) {
	pp, err := p.PurePath.WithSuffix(suffix)
	if err != nil {
		return Path{}, err
	}
	return copyPathWithPurePath(p, pp), nil
}

// Join joins the current object's path with the given elements and returns
// the resulting Path object.
func (p Path) Join(paths ...string) Path {
	return copyPathWithPurePath(p, p.PurePath.Join(paths...))
}

// JoinPath is the same as Join() except it accepts a path object
func (p Path) JoinPath(paths ...Path) Path {
	spaths := make([]string, 0, len(paths))
	for _, p := range paths {
		spaths = append(spaths, p.String())
	}
	return p.Join(spaths...)
}

// Parent returns the Path object of the parent directory.
func (p Path) Parent() Path {
	return copyPathWithPurePath(p, p.PurePath.Parent())
}

// Parents returns a list of Path objects for each parent directory.
func (p Path) Parents() []Path {
	pparents := p.PurePath.Parents()
	var parents []Path
	for _, parent := range pparents {
		parents = append(parents, copyPathWithPurePath(p, parent))
	}
	return parents
}

// RelativeTo computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p Path) RelativeTo(others ...string) (Path, error) {
	pp, err := p.PurePath.RelativeTo(others...)
	if err != nil {
		return Path{}, err
	}
	return copyPathWithPurePath(p, pp), nil
}

// RelativeToPath computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p Path) RelativeToPath(others ...Path) (Path, error) {
	othersStr := make([]string, 0, len(others))
	for _, p := range others {
		othersStr = append(othersStr, p.String())
	}
	pp, err := p.PurePath.RelativeTo(othersStr...)
	if err != nil {
		return Path{}, err
	}
	return copyPathWithPurePath(p, pp), nil
}

// -----------------------------------------------------------------------------
//
// pathlib.Path-like methods
//
// -----------------------------------------------------------------------------

// Readlink returns the target path of a symlink.
//
// This will fail if the underlying afero filesystem does not implement
// afero.LinkReader.
func (p Path) Readlink() (Path, error) {
	linkReader, ok := p.Fs().(afero.LinkReader)
	if !ok {
		return Path{}, p.doesNotImplementErr("afero.LinkReader")
	}

	resolvedPathStr, err := linkReader.ReadlinkIfPossible(p.String())
	if err != nil {
		return Path{}, err
	}
	return copyPathWithPaths(p, resolvedPathStr), nil
}

func resolveIfSymlink(path Path) (Path, bool, error) {
	isSymlink, err := path.IsSymlink()
	if err != nil {
		return path, isSymlink, err
	}
	if isSymlink {
		resolvedPath, err := path.Readlink()
		if err != nil {
			// Return the path unchanged on errors
			return path, isSymlink, err
		}
		return resolvedPath, isSymlink, nil
	}
	return path, isSymlink, nil
}

func resolveAllHelper(path Path) (Path, error) {
	parts := path.Parts()

	for i := 0; i < len(parts); i++ {
		rightOfComponent := parts[i+1:]
		upToComponent := parts[:i+1]

		componentPath := copyPathWithPaths(path, upToComponent...)
		resolved, isSymlink, err := resolveIfSymlink(componentPath)
		if err != nil {
			return path, err
		}

		if isSymlink {
			if resolved.IsAbsolute() {
				return resolveAllHelper(resolved.Join(rightOfComponent...))
			}
			return resolveAllHelper(componentPath.Parent().JoinPath(resolved).Join(rightOfComponent...))
		}
	}

	// If we get through the entire iteration above, that means no component was a symlink.
	// Return the argument.
	return path, nil
}

// ResolveAll canonicalizes the path by following every symlink in
// every component of the given path recursively. The behavior
// should be identical to the `readlink -f` command from POSIX OSs.
// This will fail if the underlying afero filesystem does not implement
// afero.LinkReader. The path will be returned unchanged on errors.
func (p Path) ResolveAll() (Path, error) {
	return resolveAllHelper(p)
}

// Lstat lstat's the path if the underlying afero filesystem supports it. If
// the filesystem does not support afero.Lstater, or if the filesystem implements
// afero.Lstater but returns false for the "lstat called" return value.
//
// A nil os.FileInfo is returned on errors.
func (p Path) Lstat() (os.FileInfo, error) {
	lStater, ok := p.Fs().(afero.Lstater)
	if !ok {
		return nil, p.doesNotImplementErr("afero.Lstater")
	}
	stat, lstatCalled, err := lStater.LstatIfPossible(p.String())
	if !lstatCalled && err == nil {
		return nil, p.lstatNotPossible()
	}
	return stat, err
}

// -----------------------------------------------------------------------------
//
// filesystem-specific methods
//
// -----------------------------------------------------------------------------

// SymlinkStr symlinks to the target location. This will fail if the underlying
// afero filesystem does not implement afero.Linker.
func (p Path) SymlinkStr(target string) error {
	return p.Symlink(copyPathWithPaths(p, target))
}

// Symlink symlinks to the target location. This will fail if the underlying
// afero filesystem does not implement afero.Linker.
func (p Path) Symlink(target Path) error {
	symlinker, ok := p.fs.(afero.Linker)
	if !ok {
		return p.doesNotImplementErr("afero.Linker")
	}

	return symlinker.SymlinkIfPossible(target.String(), p.String())
}

// -----------------------------------------------------------------------------
//
// additional methods
//
// -----------------------------------------------------------------------------

// IsFile returns true if the given path is a file.
func (p Path) IsFile() (bool, error) {
	fileInfo, err := p.Stat()
	if err != nil {
		return false, err
	}
	return IsFile(fileInfo.Mode()), nil
}

// IsFile returns whether or not the file described by the given
// os.FileMode is a regular file.
func IsFile(mode os.FileMode) bool {
	return mode.IsRegular()
}

// IsSymlink returns true if the given path is a symlink.
// Fails if the filesystem doesn't implement afero.Lstater.
func (p Path) IsSymlink() (bool, error) {
	fileInfo, err := p.Lstat()
	if err != nil {
		return false, err
	}
	return IsSymlink(fileInfo.Mode()), nil
}

// IsSymlink returns true if the file described by the given
// os.FileMode describes a symlink.
func IsSymlink(mode os.FileMode) bool {
	return mode&os.ModeSymlink != 0
}

// DeepEquals returns whether or not the path pointed to by other
// has the same resolved filepath as self.
func (p Path) DeepEquals(other Path) (bool, error) {
	selfResolved, err := p.ResolveAll()
	if err != nil {
		return false, err
	}
	otherResolved, err := other.ResolveAll()
	if err != nil {
		return false, err
	}

	return selfResolved.Clean().Equals(otherResolved.Clean()), nil
}

// Equals returns whether or not the object's path is identical
// to other's, in a shallow sense. It simply checks for equivalence
// in the unresolved Paths() of each object.
func (p Path) Equals(other Path) bool {
	return p.String() == other.String()
}

// GetLatest returns the file or directory that has the most recent mtime. Only
// works if this path is a directory and it exists. If the directory is empty,
// an ErrDirectoryEmpty will be returned.
func (p Path) GetLatest() (Path, error) {
	files, err := p.ReadDir()
	if err != nil {
		return Path{}, err
	}
	if len(files) == 0 {
		return Path{}, ErrDirectoryEmpty
	}

	greatestFileSeen := p.Join(files[0].Name())
	for i := 1; i < len(files)-1; i++ {
		file := files[i]
		greatestMtime, err := greatestFileSeen.Mtime()
		if err != nil {
			return Path{}, err
		}

		thisMtime, err := file.Mtime()
		// There is a possible race condition where the file is deleted after
		// our call to ReadDir. We throw away the error if it isn't
		// os.ErrNotExist
		if err != nil && !os.IsNotExist(err) {
			return Path{}, err
		}
		if thisMtime.After(greatestMtime) {
			greatestFileSeen = p.Join(file.Name())
		}
	}

	return greatestFileSeen, nil
}

// Glob returns all matches of pattern relative to this object's path.
func (p Path) Glob(pattern string) ([]Path, error) {
	pattern = strings.Join([]string{p.String(), pattern}, "/")
	matches, err := afero.Glob(p.fs, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to glob: %w", err)
	}

	pathMatches := []Path{}
	for _, match := range matches {
		pathMatches = append(pathMatches, copyPathWithPaths(p, match))
	}
	return pathMatches, nil
}

// Clean returns a new object that is a lexically-cleaned
// version of Path.
func (p Path) Clean() Path {
	return copyPathWithPaths(p, filepath.Clean(p.String()))
}

// Mtime returns the modification time of the given path.
func (p Path) Mtime() (time.Time, error) {
	stat, err := p.Stat()
	if err != nil {
		return time.Time{}, err
	}
	return Mtime(stat)
}

// Mtime returns the mtime described in the given os.FileInfo object
func Mtime(fileInfo os.FileInfo) (time.Time, error) {
	return fileInfo.ModTime(), nil
}

// Size returns the size of the object. Fails if the object doesn't exist.
func (p Path) Size() (int64, error) {
	stat, err := p.Stat()
	if err != nil {
		return 0, err
	}
	return Size(stat), nil
}

// Size returns the size described by the os.FileInfo. Before you say anything,
// yes... you could just do fileInfo.Size(). This is purely a convenience function
// to create API consistency.
func Size(fileInfo os.FileInfo) int64 {
	return fileInfo.Size()
}
