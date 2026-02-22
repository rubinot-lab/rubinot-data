package validation

import "fmt"

const (
	ErrorFlareSolverrConnection     = 20001
	ErrorFlareSolverrNon200         = 20002
	ErrorCloudflareChallengePresent = 20003
	ErrorEntityNotFound             = 20004
	ErrorUpstreamMaintenanceMode    = 20005
	ErrorUpstreamForbidden          = 20006
	ErrorUpstreamUnknown            = 20007
	ErrorFlareSolverrTimeout        = 20008
)

type Error struct {
	code    int
	message string
	cause   error
}

func NewError(code int, message string, cause error) Error {
	return Error{
		code:    code,
		message: message,
		cause:   cause,
	}
}

func (e Error) Error() string {
	if e.message != "" {
		return e.message
	}
	if e.cause != nil {
		return e.cause.Error()
	}
	return fmt.Sprintf("validation error %d", e.code)
}

func (e Error) Code() int {
	return e.code
}

func (e Error) Unwrap() error {
	return e.cause
}
