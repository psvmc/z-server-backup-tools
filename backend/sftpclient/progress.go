package sftpclient

import "io"

// ProgressFunc is called while copying; written is bytes so far, total may be 0 if unknown.
type ProgressFunc func(written, total int64)

type progressWriter struct {
	w       io.Writer
	written int64
	total   int64
	onProg  ProgressFunc
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n, err := pw.w.Write(p)
	if n > 0 {
		pw.written += int64(n)
		if pw.onProg != nil {
			pw.onProg(pw.written, pw.total)
		}
	}
	return n, err
}
