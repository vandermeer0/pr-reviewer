// Package usecase содержит доменные ошибки
package usecase

// ErrorCode - доменный код ошибки
type ErrorCode string

const (
	// ErrorCodeTeamExists возвращается когда команда с таким именем уже существует
	ErrorCodeTeamExists ErrorCode = "TEAM_EXISTS"
	// ErrorCodePRExists возвращается когда PR с таким идентификатором уже существует
	ErrorCodePRExists ErrorCode = "PR_EXISTS"
	// ErrorCodePRMerged возвращается когда операция запрещена для слитого PR
	ErrorCodePRMerged ErrorCode = "PR_MERGED"
	// ErrorCodeNotAssigned возвращается когда пользователь не назначен ревьювером этого PR
	ErrorCodeNotAssigned ErrorCode = "NOT_ASSIGNED"
	// ErrorCodeNoCandidate возвращается когда нет подходящего кандидата на замену ревьювера
	ErrorCodeNoCandidate ErrorCode = "NO_CANDIDATE"
	// ErrorCodeNotFound возвращается когда сущность не найдена
	ErrorCodeNotFound ErrorCode = "NOT_FOUND"
)

// DomainError представляет доменную ошибку с кодом и сообщением
type DomainError struct {
	Code    ErrorCode
	Message string
}

// Error реализует интерфейс error
func (e *DomainError) Error() string {
	return string(e.Code) + ": " + e.Message
}

// NewTeamExistsError создаёт ошибку с кодом ErrorCodeTeamExists
func NewTeamExistsError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeTeamExists,
		Message: msg,
	}
}

// NewPRExistsError создаёт ошибку с кодом ErrorCodePRExists
func NewPRExistsError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodePRExists,
		Message: msg,
	}
}

// NewPRMergedError создаёт ошибку с кодом ErrorCodePRMerged
func NewPRMergedError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodePRMerged,
		Message: msg,
	}
}

// NewNotAssignedError создаёт ошибку с кодом ErrorCodeNotAssigned
func NewNotAssignedError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeNotAssigned,
		Message: msg,
	}
}

// NewNoCandidateError создаёт ошибку с кодом ErrorCodeNoCandidate
func NewNoCandidateError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeNoCandidate,
		Message: msg,
	}
}

// NewNotFoundError создаёт ошибку с кодом ErrorCodeNotFound
func NewNotFoundError(msg string) *DomainError {
	return &DomainError{
		Code:    ErrorCodeNotFound,
		Message: msg,
	}
}
