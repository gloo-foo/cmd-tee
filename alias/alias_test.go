package alias_test

import (
	"bytes"
	"slices"
	"testing"

	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-tee"
	tee "github.com/gloo-foo/cmd-tee/alias"
	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/testable"
)

// The alias package re-exports the constructor and flag constants under
// unprefixed names. A mis-wired re-export (Append bound to Truncate, or Tee
// bound to the wrong function) compiles cleanly, so only behavior can prove the
// wiring. Each test exercises one re-export and asserts the output it produces.

// TestAlias_TeePassesThroughAndWrites proves tee.Tee forwards the stream and
// writes the named file — the constructor re-export is correctly wired.
func TestAlias_TeePassesThroughAndWrites(t *testing.T) {
	fs := afero.NewMemMapFs()
	lines, err := testable.TestLines(tee.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs}), "a\nb\n")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(lines, []string{"a", "b"}) {
		t.Fatalf("passthrough: got %q, want [a b]", lines)
	}
	if got := mustRead(t, fs, "out.txt"); !bytes.Equal(got, []byte("a\nb\n")) {
		t.Fatalf("file: got %q, want %q", got, "a\nb\n")
	}
}

// TestAlias_AppendAppends proves tee.Append re-exports the -a (append) flag:
// pre-existing content is preserved.
func TestAlias_AppendAppends(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "out.txt", []byte("first\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := testable.TestLines(tee.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs}, tee.Append), "second\n"); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, fs, "out.txt"); !bytes.Equal(got, []byte("first\nsecond\n")) {
		t.Fatalf("append: got %q, want %q", got, "first\nsecond\n")
	}
}

// TestAlias_TruncateOverwrites proves tee.Truncate re-exports the default
// (truncate) flag: pre-existing content is discarded.
func TestAlias_TruncateOverwrites(t *testing.T) {
	fs := afero.NewMemMapFs()
	if err := afero.WriteFile(fs, "out.txt", []byte("stale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := testable.TestLines(tee.Tee(gloo.File("out.txt"), command.TeeFs{Fs: fs}, tee.Truncate), "fresh\n"); err != nil {
		t.Fatal(err)
	}
	if got := mustRead(t, fs, "out.txt"); !bytes.Equal(got, []byte("fresh\n")) {
		t.Fatalf("truncate: got %q, want %q", got, "fresh\n")
	}
}

func mustRead(t *testing.T, fs afero.Fs, name string) []byte {
	t.Helper()
	data, err := afero.ReadFile(fs, name)
	if err != nil {
		t.Fatalf("read %q: %v", name, err)
	}
	return data
}
