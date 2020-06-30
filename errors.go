package main

import "fmt"

type ErrFailedCommandExecution struct {
	Ignore      bool
	CommandType CommandType
	Name        string
	Message     string
}

func IsIgnorableErrFailedCommandExecution(err error) bool {
	if e, ok := err.(ErrFailedCommandExecution); ok {
		return e.Ignore
	}
	return false
}

func IsErrFailedCommandExecution(err error) bool {
	_, ok := err.(ErrFailedCommandExecution)
	return ok
}

func (e ErrFailedCommandExecution) Error() string {
	return fmt.Sprintf("%s %s: %s", e.CommandType, e.Name, e.Message)
}
