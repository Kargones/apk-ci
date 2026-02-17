package constants

import "os"

// Directory permission constants.
const (
	// DirPermStandard is the standard directory permission (owner rwx, group r-x).
	DirPermStandard os.FileMode = 0750

	// DirPermExec is the directory permission with world read+execute (owner rwx, group r-x, other r-x).
	DirPermExec os.FileMode = 0755

	// DirPermPrivate is the directory permission for private directories (owner rwx only).
	DirPermPrivate os.FileMode = 0700
)

// File permission constants.
const (
	// FilePermReadWrite is the standard file permission (owner rw, group r, other r).
	FilePermReadWrite os.FileMode = 0644

	// FilePermPrivate is the file permission for sensitive files (owner rw only).
	FilePermPrivate os.FileMode = 0600
)
