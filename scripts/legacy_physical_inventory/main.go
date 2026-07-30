// Este fichero valida la entrada humana y coordina un único censo acotado.
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

const (
	defaultMaxEntries          = 2_000_000
	defaultMaxDirectoryEntries = 100_000
	defaultMaxDepth            = 256
	defaultMaxPathBytes        = int64(64 << 10)
	defaultMaxOutputBytes      = int64(16 << 30)
	defaultMaxHashBytes        = int64(64 << 30)
	defaultMaxFileBytes        = int64(2 << 30)
	defaultTimeout             = 2 * time.Hour
	maxInputArguments          = 4_096
	maxInputBytes              = int64(4 << 20)
	maxRootInputs              = 1_024
	maxDeniedInputs            = 2_048
)

type repeatedFlag []string

func (values *repeatedFlag) String() string { return strings.Join(*values, ",") }
func (values *repeatedFlag) Set(value string) error {
	if strings.TrimSpace(value) == "" {
		return errors.New("empty_flag_value")
	}
	*values = append(*values, value)
	return nil
}
func main() {
	if err := executeCommand(os.Args[1:], os.Stdout); err != nil {
		writeCommandError(os.Stderr, err)
		os.Exit(1)
	}
}
func writeCommandError(output io.Writer, err error) {
	fmt.Fprintf(output, "%s [%s]\n", spanishText("error.prefix"), publicErrorCode(err))
}
func executeCommand(arguments []string, output io.Writer) error {
	return executeCommandStarted(arguments, output, time.Now())
}
func executeCommandStarted(arguments []string, output io.Writer, started time.Time) error {
	if err := validateInputEnvelope(arguments); err != nil {
		return err
	}
	var rawRoots repeatedFlag
	var rawDenied repeatedFlag
	var opts options
	var help bool
	flags := flag.NewFlagSet("legacy_physical_inventory", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.Var(&rawRoots, "root", spanishText("flag.root"))
	flags.Var(&rawDenied, "deny", spanishText("flag.deny"))
	flags.StringVar(&opts.jsonlPath, "jsonl", "", spanishText("flag.jsonl"))
	flags.StringVar(&opts.manifestPath, "manifest", "", spanishText("flag.manifest"))
	flags.Int64Var(&opts.budget.maxEntries, "max-entries", defaultMaxEntries, spanishText("flag.max_entries"))
	flags.Int64Var(&opts.budget.maxDirectoryEntries, "max-directory-entries", defaultMaxDirectoryEntries, spanishText("flag.max_directory_entries"))
	flags.IntVar(&opts.budget.maxDepth, "max-depth", defaultMaxDepth, spanishText("flag.max_depth"))
	flags.Int64Var(&opts.budget.maxPathBytes, "max-path-bytes", defaultMaxPathBytes, spanishText("flag.max_path_bytes"))
	flags.Int64Var(&opts.budget.maxOutputBytes, "max-output-bytes", defaultMaxOutputBytes, spanishText("flag.max_output_bytes"))
	flags.Int64Var(&opts.budget.maxHashBytes, "max-hash-bytes", defaultMaxHashBytes, spanishText("flag.max_hash_bytes"))
	flags.Int64Var(&opts.budget.maxFileBytes, "max-file-bytes", defaultMaxFileBytes, spanishText("flag.max_file_bytes"))
	flags.DurationVar(&opts.budget.timeout, "timeout", defaultTimeout, spanishText("flag.timeout"))
	flags.BoolVar(&help, "help", false, spanishText("flag.help"))
	flags.BoolVar(&help, "h", false, spanishText("flag.help"))
	if err := flags.Parse(arguments); err != nil {
		return errors.Join(errInvalidInput, err)
	}
	if help {
		renderSpanishHelp(output)
		return nil
	}
	if flags.NArg() != 0 {
		return errUnexpectedArguments
	}
	if len(rawRoots) > maxRootInputs || len(rawDenied) > maxDeniedInputs {
		return errors.Join(errInvalidInput, errors.New("too_many_inputs"))
	}
	var err error
	opts.started = started
	opts.roots, err = parseRoots(rawRoots)
	if err == nil {
		err = applyDenied(opts.roots, rawDenied)
	}
	if err == nil {
		err = validateOptions(&opts)
	}
	if err == nil {
		err = run(opts)
	}
	return err
}
func validateInputEnvelope(arguments []string) error {
	if len(arguments) > maxInputArguments {
		return errors.Join(errInvalidInput, errors.New("too_many_arguments"))
	}
	var inputBytes int64
	for _, argument := range arguments {
		inputBytes += int64(len(argument))
		if inputBytes > maxInputBytes {
			return errors.Join(errInvalidInput, errors.New("arguments_too_large"))
		}
	}
	return nil
}
