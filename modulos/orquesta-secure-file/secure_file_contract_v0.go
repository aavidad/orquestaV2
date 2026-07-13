package orquestasecurefile

import "errors"

var (
	ErrInvalidPathV0              = errors.New("secure_file_invalid_path")
	ErrUnsafeDirectoryV0          = errors.New("secure_file_unsafe_directory")
	ErrUnsafeFileV0               = errors.New("secure_file_unsafe_file")
	ErrAnonymousFileUnsupportedV0 = errors.New("secure_file_anonymous_file_unsupported")
	ErrUnsupportedPlatformV0      = errors.New("secure_file_unsupported_platform")
)

// DirectoryOptionsV0 defines a descriptor-only traversal. Missing components
// are created relative to the already verified parent descriptor.
type DirectoryOptionsV0 struct {
	Create     bool
	CreateMode uint32
	FinalMode  uint32
}

// FileOptionsV0 always requires a regular, single-link file. ExactMode may be
// zero for agent-produced files; those still must be owned by the runtime UID
// and not writable by group or other.
type FileOptionsV0 struct {
	MaxBytes  int64
	ExactMode uint32
}
