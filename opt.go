package command

// teeAppendFlag selects append vs. truncate when writing named files (-a).
type teeAppendFlag bool

const (
	// TeeAppend (-a) appends to each named file instead of truncating it.
	TeeAppend teeAppendFlag = true
	// TeeTruncate (default) truncates each named file before writing.
	TeeTruncate teeAppendFlag = false
)
