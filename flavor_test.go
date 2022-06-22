package pathlib

// Most of the test inputs are taken from the Python pathlib test suite. For
// those the credit goes to the creators of Python. Source:
// https://github.com/python/cpython/blob/22fed605e096eb74f3aa33f6d25aee76fdc2a3fa/Lib/test/test_pathlib.py

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPosixFlavor_SplitRoot(t *testing.T) {
	assert := assert.New(t)
	flavor := newPosixFlavor()
	tests := []struct {
		path     string
		expected []string
	}{
		{"", []string{"", "", ""}},
		{"a", []string{"", "", "a"}},
		{"a/b", []string{"", "", "a/b"}},
		{"a/b/", []string{"", "", "a/b/"}},
		{"/a", []string{"", "/", "a"}},
		{"/a/b", []string{"", "/", "a/b"}},
		{"/a/b/", []string{"", "/", "a/b/"}},
		// The root is collapsed when there are redundant slashes
		// except when there are exactly two leading slashes, which
		// is a special case in POSIX
		{"//a", []string{"", "//", "a"}},
		{"///a", []string{"", "/", "a"}},
		{"///a/b", []string{"", "/", "a/b"}},
		// Paths which look like NT paths aren"t treated specially
		{"c:/a/b", []string{"", "", "c:/a/b"}},
		{"\\/a/b", []string{"", "", "\\/a/b"}},
		{"\\a\\b", []string{"", "", "\\a\\b"}},
	}
	for _, test := range tests {
		drive, root, rel := flavor.SplitRoot(test.path)
		assert.Equal(test.expected, []string{drive, root, rel})
	}
}

func TestWindowsFlavor_SplitRoot(t *testing.T) {
	assert := assert.New(t)
	flavor := newWindowsFlavor()
	tests := []struct {
		path     string
		expected []string
	}{
		{"", []string{"", "", ""}},
		{"a", []string{"", "", "a"}},
		{"a\\b", []string{"", "", "a\\b"}},
		{"\\a", []string{"", "\\", "a"}},
		{"\\a\\b", []string{"", "\\", "a\\b"}},
		{"c:a\\b", []string{"c:", "", "a\\b"}},
		{"c:\\a\\b", []string{"c:", "\\", "a\\b"}},
		// Redundant slashes in the root are collapsed
		{"\\\\a", []string{"", "\\", "a"}},
		{"\\\\\\a/b", []string{"", "\\", "a/b"}},
		{"c:\\\\a", []string{"c:", "\\", "a"}},
		{"c:\\\\\\a/b", []string{"c:", "\\", "a/b"}},
		// Valid UNC paths.
		{"\\\\a\\b", []string{"\\\\a\\b", "\\", ""}},
		{"\\\\a\\b\\", []string{"\\\\a\\b", "\\", ""}},
		{"\\\\a\\b\\c\\d", []string{"\\\\a\\b", "\\", "c\\d"}},
		// These are non-UNC paths (according to ntpath.py and test_ntpath).
		// However, command.com says such paths are invalid, so it"s
		// difficult to know what the right semantics are.
		{"\\\\\\a\\b", []string{"", "\\", "a\\b"}},
		{"\\\\a", []string{"", "\\", "a"}},
	}
	for _, test := range tests {
		drive, root, rel := flavor.SplitRoot(test.path)
		assert.Equal(test.expected, []string{drive, root, rel})
	}
}
