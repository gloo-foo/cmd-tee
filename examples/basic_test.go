package tee_test

import (
	"fmt"
	"strings"

	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-tee"
	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/framework/patterns"
)

// ExampleTee_basic shows tee passing input straight through to stdout.
func ExampleTee_basic() {
	// echo "Hello World" | tee
	patterns.MustRun(
		command.Tee(strings.NewReader("Hello World")),
	)
	// Output:
	// Hello World
}

// ExampleTee_file shows tee writing each line to a named file while also
// passing it through to stdout. The file lives on an in-memory filesystem.
func ExampleTee_file() {
	fs := afero.NewMemMapFs()
	// printf 'one\ntwo\n' | tee out.txt
	patterns.MustRun(
		command.Tee(
			strings.NewReader("one\ntwo"),
			gloo.File("out.txt"),
			command.TeeFs{Fs: fs},
		),
	)
	data, _ := afero.ReadFile(fs, "out.txt")
	fmt.Printf("file: %q\n", data)
	// Output:
	// one
	// two
	// file: "one\ntwo\n"
}
