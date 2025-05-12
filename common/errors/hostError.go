package commonErrors

import (
	"errors"

	"git.antanst.com/antanst/xerrors"
)

type HostError struct {
	xerrors.XError
}

func IsHostError(err error) bool {
	var temp *HostError
	return errors.As(err, &temp)
}

func NewHostError(err error) error {
	xerr := xerrors.XError{
		UserMsg: "",
		Code:    0,
		Err:     err,
		IsFatal: false,
	}

	return &HostError{
		xerr,
	}
}
