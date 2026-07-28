package model

// LocalPartFile is one backup part zip on disk (complete or in-progress download).
type LocalPartFile struct {
	Name      string `json:"name"`
	Path      string `json:"path"`
	SizeBytes int64  `json:"sizeBytes"`
	State     string `json:"state"` // downloaded | downloading
}

// LocalPartListing lists part zips under the configured local_dir.
type LocalPartListing struct {
	LocalDir string          `json:"localDir"`
	Files    []LocalPartFile `json:"files"`
}
