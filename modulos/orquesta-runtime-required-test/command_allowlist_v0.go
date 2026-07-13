package orquestaruntimerequiredtest

import "os"

// commandAllowlistResolutionV0 carries the parser output and the immutable
// executable identity produced by allowedCommandIdentityRegistryV0. The
// registry is the sole authority: no map or PATH lookup may produce this value.
type commandAllowlistResolutionV0 struct {
	Tokens        []string
	CommandPath   string
	identity      *allowedCommandIdentityV0
	executionFile *os.File
}

type commandAllowlistResolutionFailureV0 uint8

const (
	commandAllowlistSyntaxFailureV0 commandAllowlistResolutionFailureV0 = iota + 1
	commandAllowlistNotAllowedFailureV0
	commandAllowlistShellFailureV0
)

type commandAllowlistResolutionErrorV0 struct {
	Failure commandAllowlistResolutionFailureV0
	Command string
	Syntax  error
}

func (err *commandAllowlistResolutionErrorV0) Error() string {
	if err.Syntax != nil {
		return err.Syntax.Error()
	}
	return err.Command
}

func commandAllowlistResolutionErrorV0Of(err error) *commandAllowlistResolutionErrorV0 {
	if resolutionErr, ok := err.(*commandAllowlistResolutionErrorV0); ok {
		return resolutionErr
	}
	return nil
}
