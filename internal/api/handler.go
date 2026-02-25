package api

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/validation"
)

type endpointResult struct {
	PayloadKey string
	Payload    any
	Sources    []string
}

type endpointHandler func(c *gin.Context) (endpointResult, error)

func handleEndpoint(handler endpointHandler) gin.HandlerFunc {
	return func(c *gin.Context) {
		result, err := handler(c)
		if err != nil {
			errorCode := resolveErrorCode(err)
			httpCode := statusCodeFromErrorCode(errorCode)
			message := resolveErrorMessage(errorCode, err)
			if httpCode == http.StatusBadRequest {
				route := c.FullPath()
				if route == "" {
					route = "unknown"
				}
				validationRejections.WithLabelValues(route, strconv.Itoa(errorCode)).Inc()
			}
			c.JSON(httpCode, errorEnvelope(httpCode, errorCode, message, result.Sources))
			return
		}

		c.JSON(http.StatusOK, successEnvelope(result.PayloadKey, result.Payload, result.Sources))
	}
}

func resolveErrorCode(err error) int {
	if err == nil {
		return 0
	}

	var validationErr validation.Error
	if errors.As(err, &validationErr) {
		return validationErr.Code()
	}

	return validation.ErrorUpstreamUnknown
}

func resolveErrorMessage(code int, err error) string {
	if code == validation.ErrorUpstreamMaintenanceMode {
		return validation.UpstreamMaintenanceMessage
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func statusCodeFromErrorCode(code int) int {
	switch {
	case (code >= 10001 && code <= 10007) ||
		(code >= 11001 && code <= 11008) ||
		(code >= 14001 && code <= 14007) ||
		(code >= 30001 && code <= 30010):
		return http.StatusBadRequest
	}

	switch code {
	case validation.ErrorEntityNotFound:
		return http.StatusNotFound
	case validation.ErrorUpstreamMaintenanceMode:
		return http.StatusServiceUnavailable
	case validation.ErrorFlareSolverrTimeout:
		return http.StatusGatewayTimeout
	case validation.ErrorFlareSolverrConnection,
		validation.ErrorFlareSolverrNon200,
		validation.ErrorCloudflareChallengePresent,
		validation.ErrorUpstreamForbidden,
		validation.ErrorUpstreamUnknown:
		return http.StatusBadGateway
	default:
		return http.StatusBadGateway
	}
}
