# Tee Command Compatibility

## Summary

Highly compatible with GNU `tee`. Both the passthrough on stdout and the bytes written to each FILE operand are byte-identical to GNU `tee` for copying, multiple files, and `-a`/`--append` (verified in the `yupsh/bin-tee` Docker integration harness against Debian coreutils).

## Key Behaviors

```bash
# Copy stdin to stdout AND to each FILE operand (identity passthrough)
$ printf 'alpha\nbeta\n' | tee out.txt
alpha
beta
$ cat out.txt
alpha
beta

# Multiple FILE operands all receive the full stream, in one pass
$ printf 'x\ny\n' | tee a.txt b.txt
x
y
# a.txt and b.txt both contain "x\ny\n"

# Append (-a/--append): preserve existing content instead of truncating
$ printf 'old\n' > log.txt
$ printf 'new\n' | tee -a log.txt
new
$ cat log.txt
old
new

# With no FILE operand, tee is a pure stdin-to-stdout passthrough
$ printf 'data\n' | tee
data
```

## Intentional Divergences

- `-i` (ignore interrupt signals) is not implemented: this framework models pipelines as streams and has no command-level signal layer, so the flag would be an unobservable no-op. Signal handling belongs to the CLI host, not the Command. For the implemented behavior — passthrough, one or more FILE operands, and `-a`/`--append` — output and file contents match GNU `tee` exactly.
