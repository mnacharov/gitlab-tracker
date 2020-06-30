package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsIgnorableErrFailedCommandExecution(t *testing.T) {
	errA := ErrFailedCommandExecution{
		Ignore:      true,
		CommandType: PreFlightCommandType,
		Name:        "TEST",
		Message:     "FooBar",
	}
	assert.Equal(t, true, IsIgnorableErrFailedCommandExecution(errA))
	errB := ErrFailedCommandExecution{
		CommandType: PreFlightCommandType,
		Name:        "TEST",
		Message:     "FooBar",
	}
	assert.Equal(t, false, IsIgnorableErrFailedCommandExecution(errB))
	errC := errors.New("failed")
	assert.Equal(t, false, IsIgnorableErrFailedCommandExecution(errC))
}

func TestIsErrCommandExecutionFailed(t *testing.T) {
	errA := ErrFailedCommandExecution{
		CommandType: PreFlightCommandType,
		Name:        "TEST",
		Message:     "FooBar",
	}
	assert.Equal(t, true, IsErrFailedCommandExecution(errA))
	errB := errors.New("failed")
	assert.Equal(t, false, IsErrFailedCommandExecution(errB))
}

func TestErrCommandExecutionFailed_Error(t *testing.T) {
	err := ErrFailedCommandExecution{
		CommandType: PreFlightCommandType,
		Name:        "TEST",
		Message:     "FooBar",
	}
	assert.Equal(t, "PreFlight TEST: FooBar", err.Error())
}
