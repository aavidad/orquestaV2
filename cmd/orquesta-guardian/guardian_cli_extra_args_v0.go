package main

import (
	"flag"
	"fmt"
)

const guardianConfigExtraArgsV0 = "guardian_config_extra_args"

func rejectGuardianExtraArgsV0(flags *flag.FlagSet) error {
	if flags.NArg() == 0 {
		return nil
	}
	return guardianConfigParseErrorV0{
		Code:  guardianConfigExtraArgsV0,
		Field: fmt.Sprintf("positional_args:%d", flags.NArg()),
	}
}
