package orquestaruncontrol

import "context"

type fakeRunControlPortV0 struct {
	state RunControlStateV0
}

var _ RunControlPortV0 = fakeRunControlPortV0{}

func (fake fakeRunControlPortV0) ReadRunControlStateV0(context.Context, RunControlReadRequestV0) (RunControlStateV0, error) {
	return fake.state, nil
}

func (fake fakeRunControlPortV0) PauseRunV0(context.Context, PauseRunCommandV0) (RunControlStateV0, error) {
	fake.state.Status = RunControlStatusPausedV0
	return fake.state, nil
}

func (fake fakeRunControlPortV0) ResumeRunV0(context.Context, ResumeRunCommandV0) (RunControlStateV0, error) {
	fake.state.Status = RunControlStatusRunningV0
	return fake.state, nil
}

func (fake fakeRunControlPortV0) StopRunV0(_ context.Context, command StopRunCommandV0) (RunControlStateV0, error) {
	fake.state.Status = RunControlStatusStopRequestedV0
	fake.state.Forced = command.Forced
	return fake.state, nil
}

func (fake fakeRunControlPortV0) CancelRunV0(_ context.Context, command CancelRunCommandV0) (RunControlStateV0, error) {
	fake.state.Status = RunControlStatusCancelRequestedV0
	fake.state.Forced = command.Forced
	return fake.state, nil
}

func (fake fakeRunControlPortV0) CompleteRunControlV0(_ context.Context, command CompleteRunControlCommandV0) (RunControlStateV0, error) {
	fake.state.Status = command.TargetStatus
	return fake.state, nil
}
