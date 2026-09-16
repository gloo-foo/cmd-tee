package tee_test

import (
	"fmt"
	"strings"

	gloo "github.com/gloo-foo/framework"
	"github.com/gloo-foo/framework/patterns"
	"github.com/spf13/afero"

	command "github.com/gloo-foo/cmd-tee"
)

// ExampleTee_basic shows tee passing input straight through to stdout.
func ExampleTee_basic() {
	// echo "Hello World" | tee
	if err := patterns.Run(
		command.Tee(strings.NewReader("Hello World")),
	); err != nil {
		panic(err)
	}
	// Output:
	// Hello World
}

// ExampleTee_file shows tee writing each line to a named file while also
// passing it through to stdout. The file lives on an in-memory filesystem.
func ExampleTee_file() {
	fs := afero.NewMemMapFs()
	// printf 'one\ntwo\n' | tee out.txt
	if err := patterns.Run(
		command.Tee(
			strings.NewReader("one\ntwo"),
			gloo.File("out.txt"),
			command.TeeFs{Fs: fs},
		),
	); err != nil {
		panic(err)
	}
	data, _ := afero.ReadFile(fs, "out.txt")
	fmt.Printf("file: %q\n", data)
	// Output:
	// one
	// two
	// file: "one\ntwo\n"
}
