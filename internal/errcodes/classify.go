package errcodes

import (
	"context"
	"errors"

	"go.mongodb.org/mongo-driver/mongo"

	"vervet/internal/shell"
)

// ClassifyError maps an error to a ClassifiedError with a structured code and
// human-readable detail. Returns UnknownError with an empty detail for nil.
func ClassifyError(err error) ClassifiedError {
	if err == nil {
		return ClassifiedError{Code: UnknownError}
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ClassifiedError{Code: QueryTimeout, Detail: err.Error()}
	}

	if errors.Is(err, context.Canceled) {
		return ClassifiedError{Code: QueryCancelled, Detail: err.Error()}
	}

	if errors.Is(err, shell.ErrShellNotFound) {
		return ClassifiedError{Code: ShellNotFound, Detail: err.Error()}
	}

	if errors.Is(err, shell.ErrQueryTimeout) {
		return ClassifiedError{Code: QueryTimeout, Detail: err.Error()}
	}

	var cmdErr mongo.CommandError
	if errors.As(err, &cmdErr) {
		return classifyCommandError(cmdErr)
	}

	var serverErr mongo.ServerError
	if errors.As(err, &serverErr) {
		if serverErr.HasErrorCode(18) {
			return ClassifiedError{Code: AuthFailed, Detail: err.Error()}
		}
		if serverErr.HasErrorCode(13) {
			return ClassifiedError{Code: NotAuthorized, Detail: err.Error()}
		}
		return ClassifiedError{Code: UnknownError, Detail: err.Error()}
	}

	if mongo.IsNetworkError(err) {
		return ClassifiedError{Code: NetworkError, Detail: err.Error()}
	}

	if mongo.IsTimeout(err) {
		return ClassifiedError{Code: QueryTimeout, Detail: err.Error()}
	}

	return ClassifiedError{Code: UnknownError, Detail: err.Error()}
}

func classifyCommandError(ce mongo.CommandError) ClassifiedError {
	switch ce.Code {
	case 18:
		return ClassifiedError{Code: AuthFailed, Detail: ce.Error()}
	case 13:
		return ClassifiedError{Code: NotAuthorized, Detail: ce.Error()}
	case 26:
		return ClassifiedError{Code: NamespaceNotFound, Detail: ce.Error()}
	default:
		return ClassifiedError{Code: UnknownError, Detail: ce.Error()}
	}
}
