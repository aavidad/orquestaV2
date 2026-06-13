package orquestaopestopicregistry

import (
	"context"
	"os/exec"
)

type ExecTopicRegistryCommandRunnerV0 struct{}

func (ExecTopicRegistryCommandRunnerV0) RunTopicRegistryCommandV0(
	ctx context.Context,
	invocation TopicRegistryCommandInvocationV0,
) (TopicRegistryCommandResultV0, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, invocation.ToolPath, invocation.Args...)
	output, err := cmd.CombinedOutput()
	result := TopicRegistryCommandResultV0{
		Stdout: string(output),
	}
	if cmd.ProcessState != nil {
		result.ExitCode = cmd.ProcessState.ExitCode()
	}
	if err != nil && result.ExitCode == 0 {
		result.ExitCode = 1
	}
	return result, err
}
