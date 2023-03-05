package pathlib

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/aisbergg/go-pathlib/internal/testutils"
	"github.com/spf13/afero"
)

var algorithms = []struct {
	name string
	alg  Algorithm
}{
	{name: "AlgorithmBasic", alg: AlgorithmBasic},
	{name: "AlgorithmDepthFirst", alg: AlgorithmDepthFirst},
}

func setupWalkTest(t *testing.T, algorithm Algorithm) *Walk {
	tmpdir, err := ioutil.TempDir("", "")
	if err != nil {
		t.FailNow()
	}
	fs := afero.NewOsFs()
	root := NewPathWithFS(fs, tmpdir)
	walk, err := NewWalk(root)
	if err != nil {
		t.FailNow()
	}
	walk.Opts.Algorithm = algorithm
	return walk
}

func teardownWalkTest(t *testing.T, walk *Walk) {
	testutils.NewRequire(t).NoError(walk.root.RemoveAll())
}

func testWalkScenario(t *testing.T, w *Walk, expCalled int, throwErr, expErr error) {
	assert := testutils.NewAssert(t)
	require := testutils.NewRequire(t)
	numCalled := 0
	walkFunc := func(path Path, info os.FileInfo, err error) error {
		numCalled++
		return throwErr
	}
	err := w.Walk(walkFunc)
	if expErr == nil {
		require.NoError(err)
	} else {
		assert.EqualError(expErr, err)
	}
	if numCalled != expCalled {
		t.Errorf("expected walk function to be called %d times, actually called %d times", expCalled, numCalled)
	}
}

func TestHello(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		require.NoError(HelloWorld(w.root))
		testWalkScenario(t, w, 1, nil, nil)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestTwoFiles(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		require.NoError(NFiles(w.root, 2))
		testWalkScenario(t, w, 2, nil, nil)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestTwoFilesNested(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		require.NoError(TwoFilesAtRootTwoInSubdir(w.root))
		testWalkScenario(t, w, 5, nil, nil)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestZeroDepth(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		w.Opts.Depth = 0
		w.Opts.FollowSymlinks = true
		require.NoError(TwoFilesAtRootTwoInSubdir(w.root))
		testWalkScenario(t, w, 3, nil, nil)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestStopWalk(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		w.Opts.Depth = 0
		w.Opts.FollowSymlinks = true
		require.NoError(TwoFilesAtRootTwoInSubdir(w.root))
		testWalkScenario(t, w, 1, ErrStopWalk, nil)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestWalkFuncErr(t *testing.T) {
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)
		w.Opts.Depth = 0
		w.Opts.FollowSymlinks = true
		require.NoError(TwoFilesAtRootTwoInSubdir(w.root))
		err := fmt.Errorf("oh no")
		testWalkScenario(t, w, 1, err, err)
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}
}

func TestPassesQuerySpecification(t *testing.T) {
	assert := testutils.NewAssert(t)
	require := testutils.NewRequire(t)
	tf := func(t *testing.T, alg Algorithm) {
		w := setupWalkTest(t, alg)
		defer teardownWalkTest(t, w)

		file := w.root.Join("file.txt")
		err := file.WriteFile([]byte("hello"))
		require.NoError(err)

		stat, err := file.Stat()
		require.NoError(err)

		// File tests
		w.Opts.VisitFiles = false
		passes, err := w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.False(passes, "specified to not visit files, but passed anyway")

		w.Opts.VisitFiles = true
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.True(passes, "specified to visit files, but didn't pass")

		w.Opts.MinimumFileSize = 100
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.False(passes, "specified large file size, but passed anyway")

		w.Opts.MinimumFileSize = 0
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.True(passes, "specified smallfile size, but didn't pass")

		// Directory tests
		dir := w.root.Join("subdir")
		require.NoError(dir.MkdirAll())

		stat, err = dir.Stat()
		require.NoError(err)

		w.Opts.VisitDirs = false
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.False(passes, "specified to not visit directories, but passed anyway")

		w.Opts.VisitDirs = true
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.True(passes, "specified to visit directories, but didn't pass")

		// Symlink tests
		symlink := w.root.Join("symlink")
		require.NoError(symlink.Symlink(file))

		stat, err = symlink.Lstat()
		require.NoError(err)

		w.Opts.VisitSymlinks = false
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.False(passes, "specified to not visit symlinks, but passed anyway")

		w.Opts.VisitSymlinks = true
		passes, err = w.passesQuerySpecification(stat)
		require.NoError(err)
		assert.True(passes, "specified to visit symlinks, but didn't pass")
	}
	for _, a := range algorithms {
		t.Run(a.name, func(t *testing.T) {
			tf(t, a.alg)
		})
	}

}

func TestDefaultWalkOpts(t *testing.T) {
	tests := []struct {
		name string
		want *WalkOpts
	}{
		{"assert defaults", &WalkOpts{
			Depth:           -1,
			Algorithm:       AlgorithmBasic,
			FollowSymlinks:  false,
			MinimumFileSize: -1,
			MaximumFileSize: -1,
			VisitFiles:      true,
			VisitDirs:       true,
			VisitSymlinks:   true,
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DefaultWalkOpts(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("DefaultWalkOpts() = %v, want %v", got, tt.want)
			}
		})
	}
}

var ConfusedWandering Algorithm = 0xBADC0DE

func TestWalk_Walk(t *testing.T) {
	type fields struct {
		Opts *WalkOpts
		root Path
	}
	type args struct {
		walkFn WalkFunc
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "Bad algoritm",
			fields: fields{
				Opts: &WalkOpts{Algorithm: ConfusedWandering},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := &Walk{
				Opts: tt.fields.Opts,
				root: tt.fields.root,
			}
			if err := w.Walk(tt.args.walkFn); (err != nil) != tt.wantErr {
				t.Errorf("Walk.Walk() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
