package core

// PatchOp represents a file-level operation produced by a backend.
type PatchOp interface {
	isPatchOp()
	Path() string
}

// WriteFile creates or overwrites a file.
type WriteFile struct {
	FilePath string
	Content  []byte
}

func (WriteFile) isPatchOp()         {}
func (w WriteFile) Path() string     { return w.FilePath }

// DeleteFile removes a file.
type DeleteFile struct {
	FilePath string
}

func (DeleteFile) isPatchOp()        {}
func (d DeleteFile) Path() string    { return d.FilePath }

// RenameFile moves a file from one path to another.
type RenameFile struct {
	From string
	To   string
}

func (RenameFile) isPatchOp()        {}
func (r RenameFile) Path() string    { return r.From }

// ApplyDiff applies a unified diff to a file.
type ApplyDiff struct {
	FilePath string
	Diff     []byte
}

func (ApplyDiff) isPatchOp()         {}
func (a ApplyDiff) Path() string     { return a.FilePath }
