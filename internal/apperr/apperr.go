package apperr

import (
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Kind string

const (
	KindBadRequest      Kind = "BAD_REQUEST"
	KindUnauthenticated Kind = "UNAUTHENTICATED"
	KindForbidden       Kind = "FORBIDDEN"
	KindNotFound        Kind = "NOT_FOUND"
	KindDuplicate       Kind = "DUPLICATE"
	KindInUse           Kind = "IN_USE"
	KindInvalidState    Kind = "INVALID_STATE"
	KindValidation      Kind = "VALIDATION_ERROR"
	KindNotImplemented  Kind = "NOT_IMPLEMENTED"
	KindInternal        Kind = "INTERNAL"
)

type Error struct {
	Kind    Kind
	Message string
	Fields  map[string]string // khusus validation
	Details any               // khusus IN_USE dll
	Err     error             // error asli, JANGAN dikirim ke client
}

func (e *Error) Error() string {
	if e.Err != nil {
		return string(e.Kind) + ": " + e.Message + ": " + e.Err.Error()
	}
	return string(e.Kind) + ": " + e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// --- constructors ---

func NotFound(msg string) *Error        { return &Error{Kind: KindNotFound, Message: msg} }
func Duplicate(msg string) *Error       { return &Error{Kind: KindDuplicate, Message: msg} }
func InvalidState(msg string) *Error    { return &Error{Kind: KindInvalidState, Message: msg} }
func Unauthenticated(msg string) *Error { return &Error{Kind: KindUnauthenticated, Message: msg} }
func NotImplemented(msg string) *Error  { return &Error{Kind: KindNotImplemented, Message: msg} }

func InUse(msg string, details any) *Error {
	return &Error{Kind: KindInUse, Message: msg, Details: details}
}

func Validation(fields map[string]string) *Error {
	return &Error{Kind: KindValidation, Message: "Data is not valid.", Fields: fields}
}

func Internal(err error) *Error {
	return &Error{Kind: KindInternal, Message: "Something went wrong on the server.", Err: err}
}

func FromDB(err error, notFoundMsg string) *Error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return NotFound(notFoundMsg)
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
		return &Error{Kind: KindDuplicate, Message: "Data is already exist.", Err: err}
	}
	return Internal(err)
}
