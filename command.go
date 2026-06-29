package command

import (
	"context"
	"io"
	"os"

	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/framework/patterns"
	"github.com/spf13/afero"
)

// TeeFs injects the filesystem tee writes named File arguments to. It defaults
// to the OS filesystem; tests supply afero.NewMemMapFs().
type TeeFs struct{ afero.Fs }

// Tee returns a Command that passes lines through unchanged while writing each
// line to every named file and io.Writer as a side effect — the Unix tee.
//
// Opts are interpreted by type:
//   - gloo.File: a path written to on the injected filesystem (truncated, or
//     appended to with TeeAppend).
//   - io.Writer: registered as an additional side-effect destination.
//   - io.Reader (and not also a Writer): used as the data source, overriding
//     the upstream input stream. This lets Tee(strings.NewReader("..."), file)
//     act as a self-sourcing pipeline element in examples.
//   - TeeAppend (-a): append to files instead of truncating them.
//   - TeeFs: the filesystem File paths resolve against (defaults to OS).
func Tee(opts ...any) gloo.Command[[]byte, []byte] {
	cfg := parse(opts)
	return cfg.command()
}

// config is the immutable result of classifying Tee's options.
type config struct {
	fs      afero.Fs
	files   []gloo.File
	writers []io.Writer
	readers []io.Reader
	append  teeAppendFlag
}

// parse classifies opts into a config, defaulting the filesystem to the OS.
func parse(opts []any) config {
	cfg := config{fs: afero.NewOsFs()}
	for _, o := range opts {
		cfg = cfg.with(o)
	}
	return cfg
}

// with returns cfg extended by one classified option.
func (c config) with(o any) config {
	switch v := o.(type) {
	case TeeFs:
		c.fs = v.Fs
	case teeAppendFlag:
		c.append = v
	case gloo.File:
		c.files = append(c.files, v)
	case io.Writer:
		c.writers = append(c.writers, v)
	case io.Reader:
		c.readers = append(c.readers, v)
	}
	return c
}

// command builds the Tee Command from the parsed config, opening every named
// file as a writer before installing the side-effect tap.
func (c config) command() gloo.Command[[]byte, []byte] {
	tap := patterns.Tap(c.sink())
	if len(c.readers) == 0 {
		return tap
	}
	return c.sourcing(tap)
}

// sourcing wraps tap so the configured readers drive the input stream, letting
// Tee self-source in examples instead of reading upstream.
func (c config) sourcing(tap gloo.Command[[]byte, []byte]) gloo.Command[[]byte, []byte] {
	return gloo.FuncCommand[[]byte, []byte](func(ctx context.Context, _ gloo.Stream[[]byte]) gloo.Stream[[]byte] {
		input := gloo.ByteReaderSource(c.readers).Stream(ctx)
		return tap.Execute(ctx, input)
	})
}

// sink returns the per-line side effect: write the line (newline-terminated) to
// every writer, lazily opening each named file on first use.
func (c config) sink() func([]byte) error {
	writers := c.writers
	opened := false
	return func(line []byte) error {
		if !opened {
			files, err := c.open()
			if err != nil {
				return err
			}
			writers = append(writers, files...)
			opened = true
		}
		return broadcast(writers, line)
	}
}

// open opens every named file for writing on the injected filesystem, honoring
// the append flag.
func (c config) open() ([]io.Writer, error) {
	writers := make([]io.Writer, 0, len(c.files))
	for _, f := range c.files {
		w, err := c.openFile(f)
		if err != nil {
			closeWriters(writers)
			return nil, err
		}
		writers = append(writers, w)
	}
	return writers, nil
}

// openFile opens one named file with truncation, or appending under TeeAppend.
func (c config) openFile(f gloo.File) (io.Writer, error) {
	w, err := c.fs.OpenFile(string(f), c.flags(), 0o644)
	if err != nil {
		return nil, ErrOpenFile.With(err, "file", string(f))
	}
	return w, nil
}

// flags is the os.OpenFile mode for a destination: create+write, then either
// append or truncate.
func (c config) flags() int {
	base := os.O_CREATE | os.O_WRONLY
	if bool(c.append) {
		return base | os.O_APPEND
	}
	return base | os.O_TRUNC
}

// broadcast writes one newline-terminated line to every writer, stopping at the
// first failure.
func broadcast(writers []io.Writer, line []byte) error {
	out := make([]byte, 0, len(line)+1)
	out = append(out, line...)
	out = append(out, '\n')
	for _, w := range writers {
		if _, err := w.Write(out); err != nil {
			return ErrWrite.With(err)
		}
	}
	return nil
}

// closeWriters best-effort closes any writers that are Closers, used to unwind
// partially-opened destinations on an open error.
func closeWriters(writers []io.Writer) {
	for _, w := range writers {
		if c, ok := w.(io.Closer); ok {
			_ = c.Close()
		}
	}
}
