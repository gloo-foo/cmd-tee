package command_test

import (
	"bytes"
	"errors"
	"os"
	"slices"
	"testing"

	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-tee"
	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/testable"
)

// readFile returns the bytes written to name on fs, failing the test on error.
func readFile(t *testing.T, fs afero.Fs, name string) []byte {
	t.Helper()
	data, err := afero.ReadFile(fs, name)
	if err != nil {
		t.Fatalf("read %q: %v", name, err)
	}
	return data
}

// TestTee_Passthrough proves tee forwards input to its output stream unchanged.
func TestTee_Passthrough(t *testing.T) {
	lines, err := testable.TestLines(command.Tee(), "hello\nworld\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(lines, []string{"hello", "world"}) {
		t.Errorf("got %q, want [hello world]", lines)
	}
}

// TestTee_WritesFileAndPassesThrough proves the dual contract: every line both
// reaches the output stream AND lands in the named file, newline-terminated.
func TestTee_WritesFileAndPassesThrough(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := command.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs})

	lines, err := testable.TestLines(cmd, "alpha\nbravo\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(lines, []string{"alpha", "bravo"}) {
		t.Errorf("passthrough: got %q, want [alpha bravo]", lines)
	}
	if got := readFile(t, fs, "out.txt"); !bytes.Equal(got, []byte("alpha\nbravo\n")) {
		t.Errorf("file: got %q, want %q", got, "alpha\nbravo\n")
	}
}

// TestTee_MultipleFiles proves every named file receives the full stream.
func TestTee_MultipleFiles(t *testing.T) {
	fs := afero.NewMemMapFs()
	cmd := command.Tee(gloo.File("a.txt"), gloo.File("b.txt"), command.TeeFs{Fs: fs})

	if _, err := testable.TestLines(cmd, "x\ny\n"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, name := range []string{"a.txt", "b.txt"} {
		if got := readFile(t, fs, name); !bytes.Equal(got, []byte("x\ny\n")) {
			t.Errorf("%s: got %q, want %q", name, got, "x\ny\n")
		}
	}
}

// TestTee_Truncate proves the default overwrites pre-existing file content.
func TestTee_Truncate(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "out.txt", []byte("stale-data\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cmd := command.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs})

	if _, err := testable.TestLines(cmd, "fresh\n"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readFile(t, fs, "out.txt"); !bytes.Equal(got, []byte("fresh\n")) {
		t.Errorf("truncate: got %q, want %q", got, "fresh\n")
	}
}

// TestTee_Append proves -a preserves pre-existing content and appends.
func TestTee_Append(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "out.txt", []byte("first\n"), 0o644); err != nil {
		t.Fatalf("seed: %v", err)
	}
	cmd := command.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs}, command.TeeAppend)

	if _, err := testable.TestLines(cmd, "second\n"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := readFile(t, fs, "out.txt"); !bytes.Equal(got, []byte("first\nsecond\n")) {
		t.Errorf("append: got %q, want %q", got, "first\nsecond\n")
	}
}

// TestTee_Writer proves an io.Writer destination receives the stream alongside
// the passthrough.
func TestTee_Writer(t *testing.T) {
	var buf bytes.Buffer
	lines, err := testable.TestLines(command.Tee(&buf), "alpha\nbravo\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !slices.Equal(lines, []string{"alpha", "bravo"}) {
		t.Errorf("passthrough: got %q", lines)
	}
	if buf.String() != "alpha\nbravo\n" {
		t.Errorf("writer: got %q, want %q", buf.String(), "alpha\nbravo\n")
	}
}

// TestTee_EmptyInput proves an empty stream produces no output and no file
// writes (the lazy open never fires).
func TestTee_EmptyInput(t *testing.T) {
	fs := afero.NewMemMapFs()
	lines, err := testable.TestLines(command.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs}), "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(lines) != 0 {
		t.Errorf("got %d lines, want 0", len(lines))
	}
	if exists, _ := afero.Exists(fs, "out.txt"); exists {
		t.Errorf("empty input must not create the file")
	}
}

// TestTee_OpenError proves an un-openable destination surfaces ErrOpenFile.
func TestTee_OpenError(t *testing.T) {
	fs := afero.NewReadOnlyFs(afero.NewMemMapFs())
	cmd := command.Tee(gloo.File("nope.txt"), command.TeeFs{Fs: fs})

	_, err := testable.TestLines(cmd, "data\n")
	if !errors.Is(err, command.ErrOpenFile) {
		t.Fatalf("got %v, want ErrOpenFile", err)
	}
}

// rejectFs opens every file except one named path, which fails. It lets a test
// open the first destination successfully and fail the second, exercising the
// partial-open unwind that closes the already-opened handle.
type rejectFs struct {
	afero.Fs
	reject string
}

func (r rejectFs) OpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	if name == r.reject {
		return nil, errBoom
	}
	return r.Fs.OpenFile(name, flag, perm)
}

// TestTee_PartialOpenUnwind proves that when a later destination fails to open,
// the error surfaces AND the earlier opened file is closed (unwind path).
func TestTee_PartialOpenUnwind(t *testing.T) {
	fs := rejectFs{Fs: afero.NewMemMapFs(), reject: "bad.txt"}
	cmd := command.Tee(gloo.File("good.txt"), gloo.File("bad.txt"), command.TeeFs{Fs: fs})

	_, err := testable.TestLines(cmd, "data\n")
	if !errors.Is(err, command.ErrOpenFile) {
		t.Fatalf("got %v, want ErrOpenFile", err)
	}
}

// failWriter is an io.Writer that always fails, used to drive the write-error
// path. It is also an io.Closer so the unwind path treats it as one.
type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errBoom }
func (failWriter) Close() error              { return nil }

const errBoom = command.Error("boom")

// TestTee_WriteError proves a destination write failure surfaces ErrWrite.
func TestTee_WriteError(t *testing.T) {
	_, err := testable.TestLines(command.Tee(failWriter{}), "data\n")
	if !errors.Is(err, command.ErrWrite) {
		t.Fatalf("got %v, want ErrWrite", err)
	}
}

// TestError_With covers the sentinel wrapper's no-cause and arg branches.
func TestError_With(t *testing.T) {
	bare := command.ErrWrite.With(nil)
	if !errors.Is(bare, command.ErrWrite) {
		t.Errorf("bare: got %v, want ErrWrite", bare)
	}
	if bare.Error() != string(command.ErrWrite) {
		t.Errorf("bare message: got %q, want %q", bare.Error(), string(command.ErrWrite))
	}
	ctx := command.ErrOpenFile.With(errBoom, "file", "x.txt")
	if !errors.Is(ctx, command.ErrOpenFile) || !errors.Is(ctx, errBoom) {
		t.Errorf("wrapped error must match both sentinels: %v", ctx)
	}
}
