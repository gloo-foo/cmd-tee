// Package alias provides unprefixed names for the tee command surface.
//
//	import tee "github.com/gloo-foo/cmd-tee/alias"
//	tee.Tee(gloo.File("out.txt"), tee.Append)
package alias

import command "github.com/gloo-foo/cmd-tee"

// Tee re-exports the constructor.
var Tee = command.Tee

// Append is the -a flag: append to named files instead of truncating.
const Append = command.TeeAppend

// Truncate is the default: truncate named files before writing.
const Truncate = command.TeeTruncate
