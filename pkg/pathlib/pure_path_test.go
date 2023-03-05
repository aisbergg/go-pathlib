package pathlib

// Most of the test inputs are taken from the Python pathlib test suite. For
// those the credit goes to the creators of Python. Source:
// https://github.com/python/cpython/blob/22fed605e096eb74f3aa33f6d25aee76fdc2a3fa/Lib/test/test_pathlib.py

import (
	"testing"

	"github.com/aisbergg/go-pathlib/internal/testutils"
)

// -----------------------------------------------------------------------------
//
// parse tests
//
// -----------------------------------------------------------------------------

func TestPurePath_ParseParts(t *testing.T) {
	assert := testutils.NewAssert(t)
	flavor := newPosixFlavor()
	sep := flavor.Separator()
	tests := []struct {
		parts    []string
		expected []string
	}{
		{[]string{}, []string{"", ""}},
		{[]string{"a"}, []string{"", "", "a"}},
		{[]string{"a/"}, []string{"", "", "a"}},
		{[]string{"a", "b"}, []string{"", "", "a", "b"}},
		// expansion
		{[]string{"a/b"}, []string{"", "", "a", "b"}},
		{[]string{"a/b/"}, []string{"", "", "a", "b"}},
		{[]string{"a", "b/c", "d"}, []string{"", "", "a", "b", "c", "d"}},
		// collapsing and stripping excess slashes
		{[]string{"a", "b//c", "d"}, []string{"", "", "a", "b", "c", "d"}},
		{[]string{"a", "b/c/", "d"}, []string{"", "", "a", "b", "c", "d"}},
		// eliminating standalone dots
		{[]string{"."}, []string{"", ""}},
		{[]string{".", ".", "b"}, []string{"", "", "b"}},
		{[]string{"a", ".", "b"}, []string{"", "", "a", "b"}},
		{[]string{"a", ".", "."}, []string{"", "", "a"}},
		// the first part is anchored
		{[]string{"/a/b"}, []string{"", sep, sep, "a", "b"}},
		{[]string{"/a", "b"}, []string{"", sep, sep, "a", "b"}},
		{[]string{"/a/", "b"}, []string{"", sep, sep, "a", "b"}},
	}
	for _, test := range tests {
		drive, root, parts := parseParts(test.parts, flavor)
		res := []string{drive, root}
		res = append(res, parts...)
		assert.Equal(test.expected, res)
	}
}

func TestPurePosixPath_ParseParts(t *testing.T) {
	assert := testutils.NewAssert(t)
	flavor := newPosixFlavor()
	tests := []struct {
		parts    []string
		expected []string
	}{
		// collapsing of excess leading slashes, except for the double-slash
		// special case
		{[]string{"//a", "b"}, []string{"", "//", "//", "a", "b"}},
		{[]string{"///a", "b"}, []string{"", "/", "/", "a", "b"}},
		{[]string{"////a", "b"}, []string{"", "/", "/", "a", "b"}},
		// paths which look like NT paths aren't treated specially
		{[]string{"c:a"}, []string{"", "", "c:a"}},
		{[]string{"c:\\a"}, []string{"", "", "c:\\a"}},
		{[]string{"\\a"}, []string{"", "", "\\a"}},
		// anchored parts
		{[]string{"a", "/b", "c"}, []string{"", "/", "/", "b", "c"}},
		{[]string{"/a", "/b", "/c"}, []string{"", "/", "/", "c"}},
	}
	for _, test := range tests {
		drive, root, parts := parseParts(test.parts, flavor)
		res := []string{drive, root}
		res = append(res, parts...)
		assert.Equal(test.expected, res)
	}
}

func TestPureWindowsPath_ParseParts(t *testing.T) {
	assert := testutils.NewAssert(t)
	flavor := newWindowsFlavor()
	tests := []struct {
		parts    []string
		expected []string
	}{
		{[]string{"c:"}, []string{"c:", "", "c:"}},
		{[]string{"c:/"}, []string{"c:", "\\", "c:\\"}},
		{[]string{"/"}, []string{"", "\\", "\\"}},
		{[]string{"c:a"}, []string{"c:", "", "c:", "a"}},
		{[]string{"c:/a"}, []string{"c:", "\\", "c:\\", "a"}},
		{[]string{"/a"}, []string{"", "\\", "\\", "a"}},
		// UNC paths.
		{[]string{"//a/b"}, []string{"\\\\a\\b", "\\", "\\\\a\\b\\"}},
		{[]string{"//a/b/"}, []string{"\\\\a\\b", "\\", "\\\\a\\b\\"}},
		{[]string{"//a/b/c"}, []string{"\\\\a\\b", "\\", "\\\\a\\b\\", "c"}},
		// Second part is anchored, so that the first part is ignored.
		{[]string{"a", "Z:b", "c"}, []string{"Z:", "", "Z:", "b", "c"}},
		{[]string{"a", "Z:/b", "c"}, []string{"Z:", "\\", "Z:\\", "b", "c"}},
		// UNC paths.
		{[]string{"a", "//b/c", "d"}, []string{"\\\\b\\c", "\\", "\\\\b\\c\\", "d"}},
		// Collapsing and stripping excess slashes.
		{[]string{"a", "Z://b//c/", "d/"}, []string{"Z:", "\\", "Z:\\", "b", "c", "d"}},
		// UNC paths.
		{[]string{"a", "//b/c//", "d"}, []string{"\\\\b\\c", "\\", "\\\\b\\c\\", "d"}},
		// Extended paths.
		{[]string{"//?/c:/"}, []string{"\\\\?\\c:", "\\", "\\\\?\\c:\\"}},
		{[]string{"//?/c:/a"}, []string{"\\\\?\\c:", "\\", "\\\\?\\c:\\", "a"}},
		{[]string{"//?/c:/a", "/b"}, []string{"\\\\?\\c:", "\\", "\\\\?\\c:\\", "b"}},
		// Extended UNC paths (format is "\\?\UNC\server\share").
		{[]string{"//?/UNC/b/c"}, []string{"\\\\?\\UNC\\b\\c", "\\", "\\\\?\\UNC\\b\\c\\"}},
		{[]string{"//?/UNC/b/c/d"}, []string{"\\\\?\\UNC\\b\\c", "\\", "\\\\?\\UNC\\b\\c\\", "d"}},
		// Second part has a root but not drive.
		{[]string{"a", "/b", "c"}, []string{"", "\\", "\\", "b", "c"}},
		{[]string{"Z:/a", "/b", "c"}, []string{"Z:", "\\", "Z:\\", "b", "c"}},
		{[]string{"//?/Z:/a", "/b", "c"}, []string{"\\\\?\\Z:", "\\", "\\\\?\\Z:\\", "b", "c"}},
	}
	for _, test := range tests {
		drive, root, parts := parseParts(test.parts, flavor)
		res := []string{drive, root}
		res = append(res, parts...)
		assert.Equal(test.expected, res)
	}
}

// -----------------------------------------------------------------------------
//
// PurePath tests
//
// -----------------------------------------------------------------------------

func PP(path ...string) PurePath {
	return NewPurePath(path...)
}

func PPP(path ...string) PurePath {
	return NewPurePosixPath(path...)
}
func PWP(path ...string) PurePath {
	return NewPureWindowsPath(path...)
}

func TestPurePath_String(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("a", PP("a").String())
	assert.Equal("a/b", PP("a/b").String())
	assert.Equal("a/b/c", PP("a/b/c").String())
	assert.Equal("/", PP("/").String())
	assert.Equal("/a/b", PP("/a/b").String())
	assert.Equal("/a/b/c", PP("/a/b/c").String())
}

func TestPurePath_AsPosix(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("a", PP("a").AsPosix())
	assert.Equal("a/b", PP("a/b").AsPosix())
	assert.Equal("a/b/c", PP("a/b/c").AsPosix())
	assert.Equal("/", PP("/").AsPosix())
	assert.Equal("/a/b", PP("/a/b").AsPosix())
	assert.Equal("/a/b/c", PP("/a/b/c").AsPosix())
}

func TestPurePath_Drive(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("", PP("a/b").Drive())
	assert.Equal("", PP("/a/b").Drive())
	assert.Equal("", PP("").Drive())
}
func TestPurePath_Root(t *testing.T) {
	assert := testutils.NewAssert(t)
	sep := PP().flavor.Separator()
	assert.Equal("", PP("").Root())
	assert.Equal("", PP("a/b").Root())
	assert.Equal(sep, PP("/").Root())
	assert.Equal(sep, PP("/a/b").Root())
}
func TestPurePath_Anchor(t *testing.T) {
	assert := testutils.NewAssert(t)
	sep := PP().flavor.Separator()
	assert.Equal("", PP("").Anchor())
	assert.Equal("", PP("a/b").Anchor())
	assert.Equal(sep, PP("/").Anchor())
	assert.Equal(sep, PP("/a/b").Anchor())
}

func TestPurePath_Name(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("", PP("").Name())
	assert.Equal("", PP(".").Name())
	assert.Equal("", PP("/").Name())
	assert.Equal("b", PP("a/b").Name())
	assert.Equal("b", PP("/a/b").Name())
	assert.Equal("b", PP("/a/b/.").Name())
	assert.Equal("b.py", PP("a/b.py").Name())
	assert.Equal("b.py", PP("/a/b.py").Name())
}

func TestPurePath_Suffix(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("", PP("").Suffix())
	assert.Equal("", PP(".").Suffix())
	assert.Equal("", PP("..").Suffix())
	assert.Equal("", PP("/").Suffix())
	assert.Equal("", PP("a/b").Suffix())
	assert.Equal("", PP("/a/b").Suffix())
	assert.Equal("", PP("/a/b/.").Suffix())
	assert.Equal(".py", PP("a/b.py").Suffix())
	assert.Equal(".py", PP("/a/b.py").Suffix())
	assert.Equal("", PP("a/.hgrc").Suffix())
	assert.Equal("", PP("/a/.hgrc").Suffix())
	assert.Equal(".rc", PP("a/.hg.rc").Suffix())
	assert.Equal(".rc", PP("/a/.hg.rc").Suffix())
	assert.Equal(".gz", PP("a/b.tar.gz").Suffix())
	assert.Equal(".gz", PP("/a/b.tar.gz").Suffix())
	assert.Equal("", PP("a/Some name. Ending with a dot.").Suffix())
	assert.Equal("", PP("/a/Some name. Ending with a dot.").Suffix())
}

func TestPurePath_Suffixes(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal([]string{}, PP("").Suffixes())
	assert.Equal([]string{}, PP(".").Suffixes())
	assert.Equal([]string{}, PP("/").Suffixes())
	assert.Equal([]string{}, PP("a/b").Suffixes())
	assert.Equal([]string{}, PP("/a/b").Suffixes())
	assert.Equal([]string{}, PP("/a/b/.").Suffixes())
	assert.Equal([]string{".py"}, PP("a/b.py").Suffixes())
	assert.Equal([]string{".py"}, PP("/a/b.py").Suffixes())
	assert.Equal([]string{}, PP("a/.hgrc").Suffixes())
	assert.Equal([]string{}, PP("/a/.hgrc").Suffixes())
	assert.Equal([]string{".rc"}, PP("a/.hg.rc").Suffixes())
	assert.Equal([]string{".rc"}, PP("/a/.hg.rc").Suffixes())
	assert.Equal([]string{".tar", ".gz"}, PP("a/b.tar.gz").Suffixes())
	assert.Equal([]string{".tar", ".gz"}, PP("/a/b.tar.gz").Suffixes())
	assert.Equal([]string{}, PP("a/Some name. Ending with a dot.").Suffixes())
	assert.Equal([]string{}, PP("/a/Some name. Ending with a dot.").Suffixes())
}
func TestPurePath_Stem(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal("", PP("").Stem())
	assert.Equal("", PP(".").Stem())
	assert.Equal("..", PP("..").Stem())
	assert.Equal("", PP("/").Stem())
	assert.Equal("b", PP("a/b").Stem())
	assert.Equal("b", PP("a/b.py").Stem())
	assert.Equal(".hgrc", PP("a/.hgrc").Stem())
	assert.Equal(".hg", PP("a/.hg.rc").Stem())
	assert.Equal("b.tar", PP("a/b.tar.gz").Stem())
	assert.Equal("Some name. Ending with a dot.", PP("a/Some name. Ending with a dot.").Stem())
}

func discErr(p PurePath, _ error) PurePath { return p }
func discVal(_ interface{}, e error) error { return e }

func TestPurePath_WithName(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal(PP("a/d.xml"), discErr(PP("a/b").WithName("d.xml")))
	assert.Equal(PP("/a/d.xml"), discErr(PP("/a/b").WithName("d.xml")))
	assert.Equal(PP("a/d.xml"), discErr(PP("a/b.py").WithName("d.xml")))
	assert.Equal(PP("/a/d.xml"), discErr(PP("/a/b.py").WithName("d.xml")))
	assert.Equal(PP("a/d.xml"), discErr(PP("a/Dot ending.").WithName("d.xml")))
	assert.Equal(PP("/a/d.xml"), discErr(PP("/a/Dot ending.").WithName("d.xml")))
	assert.Error(discVal(PP("").WithName("d.xml")))
	assert.Error(discVal(PP(".").WithName("d.xml")))
	assert.Error(discVal(PP("/").WithName("d.xml")))
	assert.Error(discVal(PP("a/b").WithName("")))
	assert.Error(discVal(PP("a/b").WithName("/c")))
	assert.Error(discVal(PP("a/b").WithName("c/")))
	assert.Error(discVal(PP("a/b").WithName("c/d")))
	// original should not change
	p := PP("a/d.xml")
	p.WithName("d.png")
	assert.Equal(PP("a/d.xml"), p)
}

func TestPurePath_WithStem(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.Equal(PP("a/d"), discErr(PP("a/b").WithStem("d")))
	assert.Equal(PP("/a/d"), discErr(PP("/a/b").WithStem("d")))
	assert.Equal(PP("a/d.py"), discErr(PP("a/b.py").WithStem("d")))
	assert.Equal(PP("/a/d.py"), discErr(PP("/a/b.py").WithStem("d")))
	assert.Equal(PP("/a/d.gz"), discErr(PP("/a/b.tar.gz").WithStem("d")))
	assert.Equal(PP("a/d"), discErr(PP("a/Dot ending.").WithStem("d")))
	assert.Equal(PP("/a/d"), discErr(PP("/a/Dot ending.").WithStem("d")))
	assert.Error(discVal(PP("").WithStem("d")))
	assert.Error(discVal(PP(".").WithStem("d")))
	assert.Error(discVal(PP("/").WithStem("d")))
	assert.Error(discVal(PP("a/b").WithStem("")))
	assert.Error(discVal(PP("a/b").WithStem("/c")))
	assert.Error(discVal(PP("a/b").WithStem("c/")))
	assert.Error(discVal(PP("a/b").WithStem("c/d")))
	// original should not change
	p := PP("a/d.xml")
	p.WithStem("x")
	assert.Equal(PP("a/d.xml"), p)
}

func TestPurePath_WithSuffix(t *testing.T) {
	assert := testutils.NewAssert(t)
	sep := PP().flavor.Separator()
	assert.Equal(PP("a/b.gz"), discErr(PP("a/b").WithSuffix(".gz")))
	assert.Equal(PP("/a/b.gz"), discErr(PP("/a/b").WithSuffix(".gz")))
	assert.Equal(PP("a/b.gz"), discErr(PP("a/b.py").WithSuffix(".gz")))
	assert.Equal(PP("/a/b.gz"), discErr(PP("/a/b.py").WithSuffix(".gz")))
	// stripping suffix
	assert.Equal(PP("a/b"), discErr(PP("a/b.py").WithSuffix("")))
	assert.Equal(PP("/a/b"), discErr(PP("/a/b").WithSuffix("")))
	// path doesn"t have a "filename" component
	assert.Error(discVal(PP("").WithSuffix(".gz")))
	assert.Error(discVal(PP(".").WithSuffix(".gz")))
	assert.Error(discVal(PP("/").WithSuffix(".gz")))
	// invalid suffix
	assert.Error(discVal(PP("a/b").WithSuffix("gz")))
	assert.Error(discVal(PP("a/b").WithSuffix("/")))
	assert.Error(discVal(PP("a/b").WithSuffix(".")))
	assert.Error(discVal(PP("a/b").WithSuffix("/.gz")))
	assert.Error(discVal(PP("a/b").WithSuffix("c/d")))
	assert.Error(discVal(PP("a/b").WithSuffix(".c/.d")))
	assert.Error(discVal(PP("a/b").WithSuffix("./.d")))
	assert.Error(discVal(PP("a/b").WithSuffix(".d/.")))
	assert.Error(discVal(PP("a/b").WithSuffix(sep + "d")))
	// original should not change
	p := PP("a/d.xml")
	p.WithSuffix(".png")
	assert.Equal(PP("a/d.xml"), p)
}

func TestPurePath_Parent(t *testing.T) {
	assert := testutils.NewAssert(t)
	// relative
	p := PP("a/b/c")
	assert.Equal(PP("a/b"), p.Parent())
	assert.Equal(PP("a"), p.Parent().Parent())
	assert.Equal(PP(), p.Parent().Parent().Parent())
	assert.Equal(PP(), p.Parent().Parent().Parent().Parent())
	// anchored
	p = PP("/a/b/c")
	assert.Equal(PP("/a/b"), p.Parent())
	assert.Equal(PP("/a"), p.Parent().Parent())
	assert.Equal(PP("/"), p.Parent().Parent().Parent())
	assert.Equal(PP("/"), p.Parent().Parent().Parent().Parent())
}

func TestPurePath_Parents(t *testing.T) {
	assert := testutils.NewAssert(t)
	// relative
	p := PP("a/b/c")
	par := p.Parents()
	assert.Equal(3, len(par))
	assert.Equal([]PurePath{PP("a/b"), PP("a"), PP(".")}, par)
	p = PP("/a/b/c")
	par = p.Parents()
	assert.Equal(3, len(par))
	assert.Equal([]PurePath{PP("/a/b"), PP("/a"), PP("/")}, par)
}

func TestPurePath_Match(t *testing.T) {
	assert := testutils.NewAssert(t)
	// simple relative pattern
	assert.True(PP("b.py").Match("b.py"))
	assert.True(PP("a/b.py").Match("b.py"))
	assert.True(PP("/a/b.py").Match("b.py"))
	assert.False(PP("a.py").Match("b.py"))
	assert.False(PP("b/py").Match("b.py"))
	assert.False(PP("/a.py").Match("b.py"))
	assert.False(PP("b.py/c").Match("b.py"))
	// wildcard relative pattern
	assert.True(PP("b.py").Match("*.py"))
	assert.True(PP("a/b.py").Match("*.py"))
	assert.True(PP("/a/b.py").Match("*.py"))
	assert.False(PP("b.pyc").Match("*.py"))
	assert.False(PP("b./py").Match("*.py"))
	assert.False(PP("b.py/c").Match("*.py"))
	// multi-part relative pattern
	assert.True(PP("ab/c.py").Match("a*/*.py"))
	assert.True(PP("/d/ab/c.py").Match("a*/*.py"))
	assert.False(PP("a.py").Match("a*/*.py"))
	assert.False(PP("/dab/c.py").Match("a*/*.py"))
	assert.False(PP("ab/c.py/d").Match("a*/*.py"))
	// absolute pattern
	assert.True(PP("/b.py").Match("/*.py"))
	assert.False(PP("b.py").Match("/*.py"))
	assert.False(PP("a/b.py").Match("/*.py"))
	assert.False(PP("/a/b.py").Match("/*.py"))
	// multi-part absolute pattern
	assert.True(PP("/a/b.py").Match("/a/*.py"))
	assert.False(PP("/ab.py").Match("/a/*.py"))
	assert.False(PP("/a/b/c.py").Match("/a/*.py"))
	// multi-part glob-style pattern. (different from Python's pathlib)
	assert.True(PP("/a/b/c.py").Match("/a/*/*.py"))
	assert.False(PP("/a/b/c.py").Match("/*/*.py"))
	assert.True(PP("/a/b/c.py").Match("./*/*.py"))
	assert.True(PP("/a/b/c.py").Match("/a/**/*.py"))
	// assert.True(PP("/a/b/c.py").Match("/a/**/b/*.py"))  // TODO: this is not supported yet
	// assert.True(PP("/a/b/c.py").Match("/**/*.py"))  // TODO: this is not supported yet
	assert.True(PP("/a/b/c.py").Match("./**/*.py"))
	assert.False(PP("/a/b/c.py").Match("**/c/*.py"))
}

func TestPurePath_RelativeTo(t *testing.T) {
	assert := testutils.NewAssert(t)
	p := PP("a/b")
	assert.Error(discVal(p.RelativeTo()))
	assert.Equal(PP("a/b"), discErr(p.RelativeToPath(PP())))
	assert.Equal(PP("a/b"), discErr(p.RelativeTo("")))
	assert.Equal(PP("b"), discErr(p.RelativeToPath(PP("a"))))
	assert.Equal(PP("b"), discErr(p.RelativeTo("a")))
	assert.Equal(PP("b"), discErr(p.RelativeTo("a/")))
	assert.Equal(PP(), discErr(p.RelativeToPath(PP("a/b"))))
	assert.Equal(PP(), discErr(p.RelativeTo("a/b")))
	// with several args
	assert.Equal(PP(), discErr(p.RelativeTo("a", "b")))
	// unrelated paths
	assert.Error(discVal(p.RelativeToPath(PP("c"))))
	assert.Error(discVal(p.RelativeToPath(PP("a/b/c"))))
	assert.Error(discVal(p.RelativeToPath(PP("a/c"))))
	assert.Error(discVal(p.RelativeToPath(PP("/a"))))
	p = PP("/a/b")
	assert.Equal(PP("a/b"), discErr(p.RelativeToPath(PP("/"))))
	assert.Equal(PP("a/b"), discErr(p.RelativeTo("/")))
	assert.Equal(PP("b"), discErr(p.RelativeToPath(PP("/a"))))
	assert.Equal(PP("b"), discErr(p.RelativeTo("/a")))
	assert.Equal(PP("b"), discErr(p.RelativeTo("/a/")))
	assert.Equal(PP(), discErr(p.RelativeToPath(PP("/a/b"))))
	assert.Equal(PP(), discErr(p.RelativeTo("/a/b")))
	// unrelated paths
	assert.Error(discVal(p.RelativeToPath(PP("/c"))))
	assert.Error(discVal(p.RelativeToPath(PP("/a/b/c"))))
	assert.Error(discVal(p.RelativeToPath(PP("/a/c"))))
	assert.Error(discVal(p.RelativeToPath(PP())))
	assert.Error(discVal(p.RelativeTo("")))
	assert.Error(discVal(p.RelativeToPath(PP("a"))))
}

func TestPurePath_IsRelativeTo(t *testing.T) {
	assert := testutils.NewAssert(t)
	p := PP("a/b")
	assert.Error(discVal(p.IsRelativeTo()))
	assert.True(p.IsRelativeToPath(PP()))
	assert.True(p.IsRelativeTo(""))
	assert.True(p.IsRelativeToPath(PP("a")))
	assert.True(p.IsRelativeTo("a/"))
	assert.True(p.IsRelativeToPath(PP("a/b")))
	assert.True(p.IsRelativeTo("a/b"))
	// with several args
	assert.True(p.IsRelativeTo("a", "b"))
	// unrelated paths
	assert.False(p.IsRelativeToPath(PP("c")))
	assert.False(p.IsRelativeToPath(PP("a/b/c")))
	assert.False(p.IsRelativeToPath(PP("a/c")))
	assert.False(p.IsRelativeToPath(PP("/a")))
	p = PP("/a/b")
	assert.True(p.IsRelativeToPath(PP("/")))
	assert.True(p.IsRelativeTo("/"))
	assert.True(p.IsRelativeToPath(PP("/a")))
	assert.True(p.IsRelativeTo("/a"))
	assert.True(p.IsRelativeTo("/a/"))
	assert.True(p.IsRelativeToPath(PP("/a/b")))
	assert.True(p.IsRelativeTo("/a/b"))
	// unrelated paths
	assert.False(p.IsRelativeToPath(PP("/c")))
	assert.False(p.IsRelativeToPath(PP("/a/b/c")))
	assert.False(p.IsRelativeToPath(PP("/a/c")))
	assert.False(p.IsRelativeToPath(PP()))
	assert.False(p.IsRelativeTo(""))
	assert.False(p.IsRelativeToPath(PP("a")))
}

func TestPurePath_Join(t *testing.T) {
	assert := testutils.NewAssert(t)
	p := PP("a/b")
	pp := p.Join("c")
	assert.Equal(PP("a/b/c"), pp)
	assert.IsType(PurePath{}, p)
	pp = p.Join("c", "d")
	assert.Equal(PP("a/b/c/d"), pp)
	pp = p.JoinPath(PP("c"))
	assert.Equal(PP("a/b/c"), pp)
	pp = p.Join("/c")
	assert.Equal(PP("/c"), pp)
}

// -----------------------------------------------------------------------------
//
// PurePosixPath tests
//
// -----------------------------------------------------------------------------

func TestPurePosixPath_IsAbsolute(t *testing.T) {
	assert := testutils.NewAssert(t)
	assert.False(PPP().IsAbsolute())
	assert.False(PPP("a").IsAbsolute())
	assert.False(PPP("a/b/").IsAbsolute())
	assert.True(PPP("/").IsAbsolute())
	assert.True(PPP("/a").IsAbsolute())
	assert.True(PPP("/a/b/").IsAbsolute())
	assert.True(PPP("//a").IsAbsolute())
	assert.True(PPP("//a/b").IsAbsolute())
}

// -----------------------------------------------------------------------------
//
// PureWindowsPath tests
//
// -----------------------------------------------------------------------------

// -----------------------------------------------------------------------------
//
// Benchmarks
//
// -----------------------------------------------------------------------------

func BenchmarkPurePath(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for i := 0; i < 200; i++ {
			p := PP("/a/b/c/d/e/f/g/h/i/j")
			_ = p.Parent().Parent().Parent().Parent().Parent()
			p2 := PP("/a/b/c")
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
