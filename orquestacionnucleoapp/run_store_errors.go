package orquestacionnucleoapp

import "errors"

type RunNotFoundErrorV0 struct {
	RunRef string
}

func (err RunNotFoundErrorV0) Error() string {
	return "run no encontrado: " + err.RunRef
}

func IsRunNotFoundErrorV0(err error) bool {
	var missing RunNotFoundErrorV0
	return errors.As(err, &missing)
}
