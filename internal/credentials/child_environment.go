package credentials

import (
	"bytes"
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
)

type ChildBinding struct {
	EnvironmentName string
	CredentialRef   CredentialRef
	Version         Version
}

type ChildEnvironmentRequest struct {
	ActorRef, RequestRef string
	OwnerRef             OwnerRef
	ScopeRef             ScopeRef
	PurposeRef           PurposeRef
	PublicEnvironment    map[string]string
	Bindings             []ChildBinding
}

type ChildOutput struct {
	Result, Diagnostic, Artifact []byte
}

// WithChildEnvironment builds an exact, non-ambient environment. Material is
// resolved only for the callback and every returned byte surface is leak-gated.
func WithChildEnvironment(
	ctx context.Context,
	store Store,
	request ChildEnvironmentRequest,
	deliver func([]string) (ChildOutput, error),
) (ChildOutput, []Receipt, error) {
	if store == nil || deliver == nil {
		return ChildOutput{}, nil, NewError(ErrorChildEnvironment, "dependency")
	}
	if err := ValidateChildEnvironmentRequest(request); err != nil {
		return ChildOutput{}, nil, err
	}

	environment := make([]string, 0, len(request.PublicEnvironment)+len(request.Bindings))
	for name, value := range request.PublicEnvironment {
		environment = append(environment, name+"="+value)
	}
	receipts := make([]Receipt, len(request.Bindings))
	materials := make([][]byte, 0, len(request.Bindings))
	var output ChildOutput
	var deliveryErr error

	var resolve func(int) error
	resolve = func(index int) error {
		if index == len(request.Bindings) {
			exactEnvironment := append([]string(nil), environment...)
			sort.Strings(exactEnvironment)
			defer clear(exactEnvironment)
			candidate, callbackErr := deliver(exactEnvironment)
			if leakErr := scanChildSurfaces(materials, candidate, callbackErr); leakErr != nil {
				deliveryErr = leakErr
			} else {
				deliveryErr = safeChildCallbackError(callbackErr)
			}
			if deliveryErr != nil {
				clear(candidate.Result)
				clear(candidate.Diagnostic)
				clear(candidate.Artifact)
				return nil
			}
			output = cloneChildOutput(candidate)
			return nil
		}

		binding := request.Bindings[index]
		use := UseRequest{
			ActorRef: request.ActorRef, RequestRef: childRequestRef(request.RequestRef, index),
			CredentialRef: binding.CredentialRef, OwnerRef: request.OwnerRef,
			ScopeRef: request.ScopeRef, PurposeRef: request.PurposeRef, Version: binding.Version,
		}
		called := 0
		var material []byte
		receipt, err := store.Use(ctx, use, func(secret Secret) error {
			called++
			if called != 1 || len(secret.material) == 0 {
				return NewError(ErrorChildEnvironment, "credential_callback")
			}
			material = secret.Bytes()
			defer clear(material)
			if bytes.IndexByte(material, 0) >= 0 {
				return NewError(ErrorChildEnvironment, "credential_value")
			}
			environment = append(environment, binding.EnvironmentName+"="+string(material))
			materials = append(materials, material)
			defer func() {
				environment[len(environment)-1] = ""
				environment = environment[:len(environment)-1]
				materials = materials[:len(materials)-1]
			}()
			return resolve(index + 1)
		})
		if err != nil {
			return err
		}
		if called != 1 || len(material) == 0 {
			return NewError(ErrorChildEnvironment, "credential_callback")
		}
		receipts[index] = receipt
		return nil
	}
	resolveErr := resolve(0)
	if resolveErr != nil {
		return ChildOutput{}, nil, resolveErr
	}
	if deliveryErr != nil {
		return ChildOutput{}, receipts, deliveryErr
	}
	return output, receipts, nil
}

func ValidateChildEnvironmentRequest(request ChildEnvironmentRequest) error {
	if err := firstError(validateRef(request.ActorRef, "actor:"), validateRef(request.RequestRef, "request:"),
		ValidateOwnerRef(request.OwnerRef), ValidateScopeRef(request.ScopeRef), ValidatePurposeRef(request.PurposeRef)); err != nil {
		return err
	}
	if len(request.Bindings) == 0 {
		return NewError(ErrorChildEnvironment, "bindings")
	}
	seen := make(map[string]struct{}, len(request.PublicEnvironment)+len(request.Bindings))
	for name, value := range request.PublicEnvironment {
		if !validEnvironmentName(name) || strings.IndexByte(value, 0) >= 0 {
			return NewError(ErrorChildEnvironment, "public_environment")
		}
		seen[name] = struct{}{}
	}
	for _, binding := range request.Bindings {
		if !validEnvironmentName(binding.EnvironmentName) {
			return NewError(ErrorChildEnvironment, "binding")
		}
		if _, duplicate := seen[binding.EnvironmentName]; duplicate {
			return NewError(ErrorChildEnvironment, "duplicate_environment")
		}
		if err := ValidateCredentialRef(binding.CredentialRef); err != nil {
			return err
		}
		seen[binding.EnvironmentName] = struct{}{}
	}
	return nil
}

func validEnvironmentName(name string) bool {
	if name == "" || len(name) > 128 || !(name[0] == '_' || name[0] >= 'A' && name[0] <= 'Z' || name[0] >= 'a' && name[0] <= 'z') {
		return false
	}
	for index := 1; index < len(name); index++ {
		char := name[index]
		if !(char == '_' || char >= 'A' && char <= 'Z' || char >= 'a' && char <= 'z' || char >= '0' && char <= '9') {
			return false
		}
	}
	return true
}

func childRequestRef(parent string, index int) string {
	return parent + ":binding:" + strconv.Itoa(index)
}

func scanChildSurfaces(materials [][]byte, output ChildOutput, callbackErr error) error {
	surfaces := [][]byte{output.Result, output.Diagnostic, output.Artifact}
	if callbackErr != nil {
		projection := []byte(callbackErr.Error())
		defer clear(projection)
		surfaces = append(surfaces, projection)
	}
	for _, material := range materials {
		secret, err := NewSecret(material)
		if err != nil {
			return err
		}
		guard, err := NewLeakGuard(secret)
		secret.Destroy()
		if err != nil {
			return err
		}
		guardSurfaces := make([]LeakSurface, len(surfaces))
		for index, surface := range surfaces {
			guardSurfaces[index] = LeakSurface{Name: "child_output", Content: surface}
		}
		err = guard.Scan(guardSurfaces)
		guard.Destroy()
		if err != nil {
			return err
		}
	}
	return nil
}

func safeChildCallbackError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, context.Canceled):
		return context.Canceled
	case errors.Is(err, context.DeadlineExceeded):
		return context.DeadlineExceeded
	}
	var credentialError *Error
	if errors.As(err, &credentialError) {
		return NewError(credentialError.Code, credentialError.Field)
	}
	return NewError(ErrorChildEnvironment, "delivery")
}

func cloneChildOutput(output ChildOutput) ChildOutput {
	return ChildOutput{
		Result: append([]byte(nil), output.Result...), Diagnostic: append([]byte(nil), output.Diagnostic...),
		Artifact: append([]byte(nil), output.Artifact...),
	}
}
