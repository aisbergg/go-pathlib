package pathlib

import (
	"strings"
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
	if len(path) > 0 && path[0:1] == pf.Separator() {
		stripped := strings.TrimLeft(path, pf.Separator())
		if len(path)-len(stripped) == 2 {
			return "", pf.Separator() + pf.Separator(), stripped
		}
		return "", pf.Separator(), stripped
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

// posixFlavor represents the Windows style path flavor.
type windowsFlavor struct{}

// newWindowsFlavor returns a new windowsFlavor.
func newWindowsFlavor() windowsFlavor {
	return windowsFlavor{}
}

// Separator returns the separator of the flavor.
func (wf windowsFlavor) Separator() string {
	return "\\"
}

// AltSeparator returns the alternative separator of the flavor.
func (wf windowsFlavor) AltSeparator() string {
	return "/"
}

// HasDrive indicates whether the flavor has a drive component.
func (wf windowsFlavor) HasDrive() bool {
	return true
}

// SplitRoot splits the given path into a drive, root and relative path
// component.
func (wf windowsFlavor) SplitRoot(path string) (string, string, string) {
	// TODO: need to implement windows flavor
	if len(path) > 0 && path[0:1] == wf.Separator() {
		stripped := strings.TrimLeft(path, wf.Separator())
		if len(path)-len(stripped) == 2 {
			return "", wf.Separator() + wf.Separator(), stripped
		}
		return "", wf.Separator(), stripped
	}
	return "", "", path
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
