package validation

import "fmt"

const (
	ErrorCharacterNameEmpty           = 10001
	ErrorCharacterNameTooShort        = 10002
	ErrorCharacterNameTooLong         = 10003
	ErrorCharacterNameInvalidFormat   = 10004
	ErrorCharacterNameRepeatedSpaces  = 10005
	ErrorCharacterNameInvalidBoundary = 10006
	ErrorCharacterNameInvalidSymbols  = 10007

	ErrorWorldDoesNotExist             = 11001
	ErrorTownDoesNotExist              = 11002
	ErrorVocationDoesNotExist          = 11003
	ErrorHighscoreCategoryDoesNotExist = 11004
	ErrorHouseStateDoesNotExist        = 11005
	ErrorHouseIDInvalid                = 11006
	ErrorHouseDoesNotExist             = 11007
	ErrorWorldIDDoesNotExist           = 11008

	ErrorGuildNameEmpty           = 14001
	ErrorGuildNameTooShort        = 14002
	ErrorGuildNameTooLong         = 14003
	ErrorGuildNameInvalidFormat   = 14004
	ErrorGuildNameRepeatedSpaces  = 14005
	ErrorGuildNameInvalidBoundary = 14006
	ErrorGuildNameInvalidSymbols  = 14007

	ErrorFlareSolverrConnection     = 20001
	ErrorFlareSolverrNon200         = 20002
	ErrorCloudflareChallengePresent = 20003
	ErrorEntityNotFound             = 20004
	ErrorUpstreamMaintenanceMode    = 20005
	ErrorUpstreamForbidden          = 20006
	ErrorUpstreamUnknown            = 20007
	ErrorFlareSolverrTimeout        = 20008

	ErrorPageOutOfBounds    = 30001
	ErrorNewsIDInvalid      = 30002
	ErrorLevelFilterInvalid = 30003
	ErrorMonthInvalid       = 30004
	ErrorYearInvalid        = 30005
	ErrorArchiveDaysInvalid = 30006
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
