package pathlib

import (
	"errors"
	"path/filepath"
	"runtime"
	"strings"
)

// PurePath represents a filesystem path and offers methods for manipulating it
// without any I/O operations.
type PurePath struct {
	parts []string
	drive string
	root  string

	// flavor defines the behavior of the path depending on the OS.
	flavor flavorer
}

// NewPurePath returns a new `PurePath` from the given path(s). Depending on the
// OS either a Windows or Posix flavored path is created.
func NewPurePath(paths ...string) PurePath {
	if runtime.GOOS == "windows" {
		return newPurePathWithFlavor(newWindowsFlavor(), paths...)
	}
	return newPurePathWithFlavor(newPosixFlavor(), paths...)
}

// NewPurePosixPath returns a new Posix flavored `PurePath` from the given
// path(s).
func NewPurePosixPath(paths ...string) PurePath {
	return newPurePathWithFlavor(newPosixFlavor(), paths...)
}

// NewPureWindowsPath returns a new Windows flavored `PurePath` from the given
// path(s).
func NewPureWindowsPath(paths ...string) PurePath {
	return newPurePathWithFlavor(newWindowsFlavor(), paths...)
}

// newPurePathWithFlavor returns a new `PurePath` from the given path(s) and
// flavor.
func newPurePathWithFlavor(flavor flavorer, paths ...string) PurePath {
	drive, root, parts := parseParts(paths, flavor)
	return PurePath{
		drive:  drive,
		root:   root,
		parts:  parts,
		flavor: flavor,
	}
}

// newPurePathFromParts returns a new `PurePath` from the given parts and
// flavor.
func newPurePathFromParts(flavor flavorer, drive, root string, parts []string) PurePath {
	return PurePath{
		drive:  drive,
		root:   root,
		parts:  parts,
		flavor: flavor,
	}
}

// parseParts parses the given path parts into drive, root, and parts.
func parseParts(paths []string, flavor flavorer) (drive string, root string, parts []string) {
	parts = make([]string, 0, 16)
	for _, part := range paths {
		if part == "" {
			continue
		}
		if flavor.AltSeparator() != "" {
			part = strings.Replace(part, flavor.AltSeparator(), flavor.Separator(), -1)
		}
		pdrive, proot, prel := flavor.SplitRoot(part)

		// replace parts, if current part is anchored
		// e.g. {"a", "Z:\b", "c"} will result in parts containing only "Z:\b" and "c"
		if pdrive != "" { // if drive is given, replace whole
			drive, root = pdrive, proot
			parts = make([]string, 0, 16)
			parts = append(parts, drive+root)
		} else if proot != "" { // if only root is given, replace parts and keep drive
			root = proot
			parts = make([]string, 0, 16)
			parts = append(parts, drive+root)
		}

		for {
			// its more efficient to search for path separator manually than
			// using strings.Split
			if i := strings.Index(prel, flavor.Separator()); i >= 0 {
				pp := prel[:i]
				if pp != "" && pp != "." {
					parts = append(parts, pp)
				}
				if i >= len(prel)-1 {
					break
				}
				prel = prel[i+1:]
			} else {
				if prel != "" && prel != "." {
					parts = append(parts, prel)
				}
				break
			}
		}
	}
	return drive, root, parts
}

// -----------------------------------------------------------------------------
//
// pathlib.PurePath-like methods
//
// -----------------------------------------------------------------------------

// AsPosix returns a string representation of the path with forward slashes.
func (p PurePath) AsPosix() string {
	return strings.ReplaceAll(p.String(), p.flavor.Separator(), "/")
}

// Drive returns the drive prefix (letter or UNC path).
func (p PurePath) Drive() string {
	return p.drive
}

// Root returns the root of the path.
func (p PurePath) Root() string {
	return p.root
}

// Anchor returns the concatenation of the drive and root component, or "".
func (p PurePath) Anchor() string {
	return p.drive + p.root
}

// Parts returns the individual components of a path
func (p PurePath) Parts() []string {
	return p.parts
}

// Name returns the string representing the final path component.
func (p PurePath) Name() string {
	if len(p.parts) == 0 ||
		((p.drive != "" || p.root != "") && len(p.parts) == 1) {
		return ""
	}
	return p.parts[len(p.parts)-1]
}

// Suffix returns the final component's last suffix including the leading dot.
// For example, in "/path/to/foo.bar", Suffix returns ".bar".
func (p PurePath) Suffix() string {
	name := p.Name()
	i := strings.LastIndex(name, ".")
	if 0 < i && i < len(name)-1 {
		return name[i:]
	}
	return ""
}

// Suffixes returns a list of the path's file extensions.
// For example, in "/path/to/foo.bar.baz", Suffixes returns
// []string{".bar", ".baz"}.
func (p PurePath) Suffixes() []string {
	name := p.Name()
	if strings.HasSuffix(name, ".") {
		return []string{}
	}
	name = strings.TrimPrefix(name, ".")
	splits := strings.Split(name, ".")[1:]
	suffixes := make([]string, 0, len(splits))
	for _, s := range splits {
		suffixes = append(suffixes, "."+s)
	}
	return suffixes
}

// Stem returns the final path component, minus any trailing suffix.
// For example, in "/path/to/foo.bar.baz", Stem returns "foo.bar".
func (p PurePath) Stem() string {
	name := p.Name()
	i := strings.LastIndex(name, ".")
	if 0 < i && i < len(name)-1 {
		return name[:i]
	}
	return name
}

// WithName returns a new path with the file name changed.
func (p PurePath) WithName(name string) (PurePath, error) {
	if p.Name() == "" {
		return PurePath{}, errors.New("path has an empty name")
	}
	drive, root, parts := parseParts([]string{name}, p.flavor)
	if name == "" ||
		name[len(name)-1:] == p.flavor.Separator() ||
		name[len(name)-1:] == p.flavor.AltSeparator() ||
		drive != "" || root != "" || len(parts) != 1 {
		return PurePath{}, errors.New("invalid name")
	}
	// need to create a array to avoid modifying the original
	parts = make([]string, len(p.parts))
	copy(parts, p.parts)
	return newPurePathFromParts(p.flavor, p.drive, p.root, append(parts[:len(parts)-1], name)), nil
}

// WithStem returns a new path with the stem changed.
func (p PurePath) WithStem(stem string) (PurePath, error) {
	return p.WithName(stem + p.Suffix())
}

// WithSuffix returns a new path with the file suffix changed. If the path has
// no suffix, the suffix is added; if the suffix is empty, the suffix is removed
// from the path.
func (p PurePath) WithSuffix(suffix string) (PurePath, error) {
	if (suffix != "" && (!strings.HasPrefix(suffix, ".") || suffix == ".")) ||
		strings.Contains(suffix, p.flavor.Separator()) ||
		(p.flavor.AltSeparator() != "" && strings.Contains(suffix, p.flavor.AltSeparator())) {
		return PurePath{}, errors.New("invalid suffix")
	}
	name := p.Name()
	if name == "" {
		return PurePath{}, errors.New("path has an empty name")
	}
	oldSuffix := p.Suffix()
	if oldSuffix == "" {
		name = name + suffix
	} else {
		name = name[:len(name)-len(oldSuffix)] + suffix
	}
	// need to create a array to avoid modifying the original
	parts := make([]string, len(p.parts))
	copy(parts, p.parts)
	return newPurePathFromParts(p.flavor, p.drive, p.root, append(parts[:len(parts)-1], name)), nil
}

// Join joins the current object's path with the given elements and returns
// the resulting Path object.
func (p PurePath) Join(paths ...string) PurePath {
	spaths := make([]string, 0, len(paths)+1)
	spaths = append(spaths, p.String())
	spaths = append(spaths, paths...)
	return newPurePathWithFlavor(p.flavor, spaths...)
}

// JoinPath is the same as Join() except it accepts a path object
func (p PurePath) JoinPath(paths ...PurePath) PurePath {
	spaths := make([]string, 0, len(paths))
	for _, p := range paths {
		spaths = append(spaths, p.String())
	}
	return p.Join(spaths...)
}

// Parent returns the Path object of the parent directory.
func (p PurePath) Parent() PurePath {
	drive, root, parts := p.drive, p.root, p.parts
	if len(parts) == 0 {
		return PurePath{
			flavor: p.flavor,
			parts:  []string{},
		}
	}
	if len(parts) == 1 && (drive != "" || root != "") {
		return p
	}
	// no need to copy parts slice, because underlying array is not modified
	return newPurePathFromParts(p.flavor, drive, root, parts[:len(parts)-1])
}

// Parents returns a list of Path objects for each parent directory.
func (p PurePath) Parents() (parents []PurePath) {
	drive, root, parts := p.drive, p.root, p.parts
	numParents := 0
	if drive != "" || root != "" {
		numParents = len(parts) - 1
	} else {
		numParents = len(parts)
	}
	parents = make([]PurePath, 0, numParents)
	for i := len(parts) - 1; i >= len(parts)-numParents; i-- {
		// no need to copy parts slice, because underlying array is not modified
		parent := newPurePathFromParts(p.flavor, drive, root, parts[:i])
		parents = append(parents, parent)
	}
	return parents
}

// RelativeTo computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p PurePath) RelativeTo(others ...string) (PurePath, error) {
	if len(others) == 0 {
		return PurePath{}, errors.New("at least one other path must be provided")
	}
	drive, root, parts := p.drive, p.root, p.parts
	var absParts []string
	if root != "" {
		absParts = make([]string, 0, len(parts)+1)
		absParts = append(absParts, drive)
		absParts = append(absParts, root)
		absParts = append(absParts, parts[1:]...)
	} else {
		absParts = parts
	}

	toPath := newPurePathWithFlavor(p.flavor, others...)
	toDrive, toRoot, toParts := toPath.Drive(), toPath.Root(), toPath.Parts()
	var toAbsParts []string
	if toRoot != "" {
		toAbsParts = make([]string, 0, len(toParts)+1)
		toAbsParts = append(toAbsParts, toDrive)
		toAbsParts = append(toAbsParts, toRoot)
		toAbsParts = append(toAbsParts, toParts[1:]...)
	} else {
		toAbsParts = toParts
	}

	casefoldComp := func(tp, op []string) bool {
		if len(op) > len(tp) {
			return false
		}
		tp = tp[:len(op)]
		l := len(tp)
		tp = p.flavor.CasefoldParts(tp)
		op = p.flavor.CasefoldParts(op)
		for i := 0; i < l; i++ {
			if tp[i] != op[i] {
				return false
			}
		}
		return true
	}
	n := len(toAbsParts)
	if n == 0 {
		if drive != "" || root != "" {
			return PurePath{}, errors.New("path is not relative")
		}
	} else if !casefoldComp(absParts, toAbsParts) {
		return PurePath{}, errors.New("path is not relative")
	}
	if n != 1 {
		root = ""
	}
	return newPurePathFromParts(p.flavor, "", root, absParts[n:]), nil
}

// RelativeToPath computes a relative version of path to the other path. For instance,
// if the object is /path/to/foo.txt and you provide /path/ as the argment, the
// returned Path object will represent to/foo.txt.
func (p PurePath) RelativeToPath(others ...PurePath) (PurePath, error) {
	if len(others) == 0 {
		return PurePath{}, errors.New("at least one other path must be provided")
	}
	if len(others) > 1 {
		othersStr := make([]string, 0, len(others))
		for _, p := range others {
			othersStr = append(othersStr, p.String())
		}
		return p.RelativeTo(othersStr...)
	}
	return p.RelativeTo(others[0].String())
}

// IsRelativeTo returns whether or not the path is relative to the other path.
func (p PurePath) IsRelativeTo(other ...string) (bool, error) {
	if len(other) == 0 {
		return false, errors.New("at least one other path must be provided")
	}
	_, err := p.RelativeTo(other...)
	return err == nil, nil
}

// IsRelativeToPath returns whether or not the path is relative to the other path.
func (p PurePath) IsRelativeToPath(other ...PurePath) (bool, error) {
	if len(other) == 0 {
		return false, errors.New("at least one other path must be provided")
	}
	_, err := p.RelativeToPath(other...)
	return err == nil, nil
}

// IsAbsolute returns whether or not the path is an absolute path. A absolute
// path has both a root and, if applicable, a drive.
func (p PurePath) IsAbsolute() bool {
	if p.root == "" {
		return false
	}
	return !p.flavor.HasDrive() || p.drive != ""
}

// Match returns whether or not the path matches the given pattern.
func (p PurePath) Match(pattern string) bool {
	cf := p.flavor.Casefold
	pattern = cf(pattern)
	patDrive, patRoot, patParts := parseParts([]string{pattern}, p.flavor)
	if len(patParts) == 0 {
		return false
	}
	if patDrive != "" && patDrive != cf(p.drive) {
		return false
	}
	if patRoot != "" && patRoot != cf(p.root) {
		return false
	}
	parts := p.flavor.CasefoldParts(p.parts)
	if patDrive != "" || patRoot != "" {
		if len(patParts) != len(parts) {
			return false
		}
		patParts = patParts[1:]
	} else if len(patParts) > len(parts) {
		return false
	}
	parti := len(parts) - 1
	for i := len(patParts) - 1; i >= 0; i-- {
		pat := patParts[i]
		part := parts[parti]
		parti--
		match, err := filepath.Match(pat, part)
		if err != nil || !match {
			return false
		}
	}
	return true
}

// -----------------------------------------------------------------------------
//
// additional methods
//
// -----------------------------------------------------------------------------

// String returns the string representation of the path
func (p PurePath) String() string {
	if p.drive != "" || p.root != "" {
		return p.drive + p.root + strings.Join(p.parts[1:], p.flavor.Separator())
	}
	if len(p.parts) == 0 {
		return "."
	}
	return strings.Join(p.parts, p.flavor.Separator())
}

// Equals returns whether or not the object's path is identical
// to other's, in a shallow sense. It simply checks for equivalence
// in the unresolved Paths() of each object.
func (p PurePath) Equals(other PurePath) bool {
	return p.String() == other.String()
}

// Clean returns a new object that is a lexically-cleaned
// version of Path.
func (p PurePath) Clean() PurePath {
	return newPurePathWithFlavor(p.flavor, filepath.Clean(p.String()))
}
