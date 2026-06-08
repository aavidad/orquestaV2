package orquestaserver

import (
	"context"
	"errors"
	"os"
	"strconv"
	"strings"
)

type ProcSelfCPUSamplerV0 struct{}

func NewProcSelfCPUSamplerV0() ProcSelfCPUSamplerV0 {
	return ProcSelfCPUSamplerV0{}
}

func (ProcSelfCPUSamplerV0) SampleSelfWatchdogCPUV0(
	ctx context.Context,
) (SelfWatchdogCPUSampleV0, error) {
	if err := ctx.Err(); err != nil {
		return SelfWatchdogCPUSampleV0{}, err
	}
	processData, err := os.ReadFile("/proc/self/stat")
	if err != nil {
		return SelfWatchdogCPUSampleV0{}, err
	}
	totalData, err := os.ReadFile("/proc/stat")
	if err != nil {
		return SelfWatchdogCPUSampleV0{}, err
	}
	processTicks, err := parseProcSelfStatTicksV0(string(processData))
	if err != nil {
		return SelfWatchdogCPUSampleV0{}, err
	}
	totalTicks, err := parseProcStatTotalTicksV0(string(totalData))
	if err != nil {
		return SelfWatchdogCPUSampleV0{}, err
	}
	return SelfWatchdogCPUSampleV0{ProcessTicks: processTicks, TotalTicks: totalTicks}, nil
}

func parseProcSelfStatTicksV0(data string) (uint64, error) {
	endComm := strings.LastIndex(data, ")")
	if endComm < 0 || endComm+2 >= len(data) {
		return 0, errors.New("proc_self_stat_invalid")
	}
	fields := strings.Fields(data[endComm+2:])
	if len(fields) <= 12 {
		return 0, errors.New("proc_self_stat_short")
	}
	utime, err := strconv.ParseUint(fields[11], 10, 64)
	if err != nil {
		return 0, errors.New("proc_self_stat_utime_invalid")
	}
	stime, err := strconv.ParseUint(fields[12], 10, 64)
	if err != nil {
		return 0, errors.New("proc_self_stat_stime_invalid")
	}
	return utime + stime, nil
}

func parseProcStatTotalTicksV0(data string) (uint64, error) {
	lines := strings.Split(data, "\n")
	if len(lines) == 0 || !strings.HasPrefix(lines[0], "cpu ") {
		return 0, errors.New("proc_stat_cpu_missing")
	}
	var total uint64
	for _, field := range strings.Fields(strings.TrimPrefix(lines[0], "cpu ")) {
		value, err := strconv.ParseUint(field, 10, 64)
		if err != nil {
			return 0, errors.New("proc_stat_cpu_invalid")
		}
		total += value
	}
	if total == 0 {
		return 0, errors.New("proc_stat_cpu_zero")
	}
	return total, nil
}
