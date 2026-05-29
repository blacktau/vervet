package errcodes

const (
	ConnectionFailed      = "connection_failed"
	AuthFailed            = "auth_failed"
	QueryTimeout          = "query_timeout"
	QuerySyntaxError      = "query_syntax_error"
	NetworkError          = "network_error"
	NotAuthorized         = "not_authorized"
	NamespaceNotFound     = "namespace_not_found"
	ShellNotFound         = "shell_not_found"
	NoDatabaseSelected    = "no_database_selected"
	QueryCancelled        = "query_cancelled"
	OIDCLoginCanceled     = "oidc_login_canceled"
	OperationNotSupported = "operation_not_supported"
	DuplicateGroupName    = "duplicate_group_name"
	UnknownError          = "unknown_error"
)

type ClassifiedError struct {
	Code   string
	Detail string
}
