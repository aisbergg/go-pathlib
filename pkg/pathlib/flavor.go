package pathlib

import (
	"strings"
	"unicode/utf8"
)

type flavorer interface {
	// Separator returns the separator of the flavor.
	Separator() string

	// AltSeparator returns the alternative separator of the flavor.
	AltSeparator() string

	// HasDrive indicates whether the flavor has a drive component.
	HasDrive() bool

	// SplitRoot splits the given path into a drive, root and relative path
	// component.
	SplitRoot(path string) (string, string, string)

	// Casefold returns the given string in a casefolded form.
	Casefold(s string) string

	// CasefoldParts returns the given parts in a casefolded form.
	CasefoldParts(parts []string) []string
}

// -----------------------------------------------------------------------------
//
// Posix Flavor
//
// -----------------------------------------------------------------------------

// posixFlavor represents the Posix style path flavor.
type posixFlavor struct{}

// newPosixFlavor returns a new PosixFlavor.
func newPosixFlavor() posixFlavor {
	return posixFlavor{}
}

// Separator returns the separator of the flavor.
func (pf posixFlavor) Separator() string {
	return "/"
}

// AltSeparator returns the alternative separator of the flavor.
func (pf posixFlavor) AltSeparator() string {
	return ""
}

// HasDrive indicates whether the flavor has a drive component.
func (pf posixFlavor) HasDrive() bool {
	return false
}

// SplitRoot splits the given path into a drive, root and relative path
// component.
func (pf posixFlavor) SplitRoot(path string) (string, string, string) {
	if len(path) > 0 && path[0:1] == "/" {
		stripped := strings.TrimLeft(path, "/")
		if len(path)-len(stripped) == 2 {
			return "", "//", stripped
		}
		return "", "/", stripped
	}
	return "", "", path
}

// Casefold returns the given string in a casefolded form.
func (pf posixFlavor) Casefold(s string) string {
	return s
}

// CasefoldParts returns the given parts in a casefolded form.
func (pf posixFlavor) CasefoldParts(parts []string) []string {
	return parts
}

// -----------------------------------------------------------------------------
//
// Windows Flavor
//
// -----------------------------------------------------------------------------

var (
	windowsSeparator    = "\\"
	windowsAltSeparator = "/"
	windowsDriveLetters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
)

// posixFlavor represents the Windows style path flavor.
type windowsFlavor struct{}

// newWindowsFlavor returns a new windowsFlavor.
func newWindowsFlavor() windowsFlavor {
	return windowsFlavor{}
}

// Separator returns the separator of the flavor.
func (wf windowsFlavor) Separator() string {
	return windowsSeparator
}

// AltSeparator returns the alternative separator of the flavor.
func (wf windowsFlavor) AltSeparator() string {
	return windowsAltSeparator
}

// HasDrive indicates whether the flavor has a drive component.
func (wf windowsFlavor) HasDrive() bool {
	return true
}

// SplitRoot splits the given path into a drive, root and relative path
// component.
func (wf windowsFlavor) SplitRoot(path string) (string, string, string) {
	var (
		sep    = '\\'
		prefix string
	)
	// check for extended-length path
	if strings.HasPrefix(path, `\\?\`) {
		if strings.HasPrefix(path[4:], `UNC\`) {
			prefix = path[:7]
			path = `\` + path[7:]
		} else {
			prefix = path[:4]
			path = path[4:]
		}
	}
	runes := []rune(path)
	first := utf8.RuneError
	if len(runes) > 0 {
		first = runes[0]
	}
	second := utf8.RuneError
	if len(runes) > 1 {
		second = runes[1]
	}
	third := utf8.RuneError
	if len(runes) > 2 {
		third = runes[2]
	}
	if second == sep && first == sep && third != sep {
		// is a UNC path:
		// vvvvvvvvvvvvvvvvvvvvv root
		// \\machine\mountpoint\directory\etc\...
		//            directory ^^^^^^^^^^^^^^
		index := runesIndexOffset(runes, sep, 2) // index of starting byte
		if index != -1 {
			index2 := runesIndexOffset(runes, sep, index+1)
			// a UNC path can't have two slashes in a row
			// (after the initial two)
			if index2 != index+1 {
				if index2 == -1 {
					index2 = len(runes)
				}
				p := ""
				if index2+1 <= len(runes) {
					p = string(runes[index2+1:])
				}
				if prefix != "" {
					return prefix + string(runes[1:index2]), string(sep), p
				}
				return string(runes[:index2]), string(sep), p
			}
		}
	}
	drv, root := "", ""
	if second == ':' && strings.IndexRune(windowsDriveLetters, first) != -1 {
		drv = string(runes[:2])
		runes = runes[2:]
		first = third
	}
	if first == sep {
		root = string(first)
		// trim left
		i := 0
		for ; i < len(runes); i++ {
			if runes[i] != sep {
				break
			}
		}
		runes = runes[i:]
	}
	return prefix + drv, root, string(runes)
}

// Casefold returns the given string in a casefolded form.
func (wf windowsFlavor) Casefold(s string) string {
	return strings.ToLower(s)
}

// CasefoldParts returns the given parts in a casefolded form.
func (wf windowsFlavor) CasefoldParts(parts []string) []string {
	for i := 0; i < len(parts); i++ {
		parts[i] = strings.ToLower(parts[i])
	}
	return parts
}

func runesIndexOffset(runes []rune, r rune, offset int) int {
	if offset >= len(runes)-1 {
		return -1
	}
	runes = runes[offset:]
	idx := -1
	for i := 0; i < len(runes); i++ {
		if runes[i] == r {
			idx = i
			break
		}
	}
	return idx + offset
}
