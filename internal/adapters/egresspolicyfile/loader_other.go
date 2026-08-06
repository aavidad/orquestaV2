//go:build !linux

package egresspolicyfile

type loaderHooks struct{}

func loadPinnedPolicy(
	_ string,
	_ string,
	_ uint32,
	_ int64,
	_ loaderHooks,
) ([]byte, error) {
	return nil, &Error{Code: CodePlatformUnsupported}
}
