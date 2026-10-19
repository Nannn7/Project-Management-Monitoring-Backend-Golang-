package response

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/yourorg/pmo-agi-be/internal/apperr"
)

// --- Envelope types ---

type SuccessResponse struct {
	Data any `json:"data"`
}

type ListResponse struct {
	Data any      `json:"data"`
	Meta MetaInfo `json:"meta"`
}

type MetaInfo struct {
	Page     int `json:"page"`
	PageSize int `json:"pageSize"`
	Total    int `json:"total"`
}

type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	Details any               `json:"details"`
}

// --- Helper functions ---

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func Success(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, SuccessResponse{Data: data})
}

func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, SuccessResponse{Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func List(w http.ResponseWriter, data any, page, pageSize, total int) {
	JSON(w, http.StatusOK, ListResponse{
		Data: data,
		Meta: MetaInfo{Page: page, PageSize: pageSize, Total: total},
	})
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, ErrorResponse{Error: ErrorDetail{
		Code:    code,
		Message: message,
		Details: nil,
	}})
}

func ValidationError(w http.ResponseWriter, fields map[string]string) {
	JSON(w, http.StatusUnprocessableEntity, ErrorResponse{Error: ErrorDetail{
		Code:    "VALIDATION_ERROR",
		Message: "Data tidak valid.",
		Fields:  fields,
		Details: nil,
	}})
}

func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", "Token tidak valid atau sudah expired.")
}

func Forbidden(w http.ResponseWriter) {
	Error(w, http.StatusForbidden, "FORBIDDEN", "Anda tidak memiliki akses.")
}

func NotFound(w http.ResponseWriter) {
	Error(w, http.StatusNotFound, "NOT_FOUND", "Data tidak ditemukan.")
}

func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "INTERNAL", "Terjadi kesalahan internal di server.")
}

func Conflict(w http.ResponseWriter, code, message string) {
	Error(w, http.StatusConflict, code, message)
}

func statusOf(k apperr.Kind) int {
	switch k {
	case apperr.KindBadRequest:
		return http.StatusBadRequest
	case apperr.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindDuplicate, apperr.KindInUse, apperr.KindInvalidState:
		return http.StatusConflict
	case apperr.KindValidation:
		return http.StatusUnprocessableEntity
	case apperr.KindNotImplemented:
		return http.StatusNotImplemented
	default:
		return http.StatusInternalServerError
	}
}

func FromError(w http.ResponseWriter, r *http.Request, err error) {
	var ae *apperr.Error
	if !errors.As(err, &ae) {
		ae = apperr.Internal(err)
	}

	if ae.Kind == apperr.KindInternal {
		log.Printf("[req=%s] internal error: %v", chimiddleware.GetReqID(r.Context()), ae.Err)
	}

	JSON(w, statusOf(ae.Kind), ErrorResponse{Error: ErrorDetail{
		Code:    string(ae.Kind),
		Message: ae.Message,
		Fields:  ae.Fields,
		Details: ae.Details,
	}})
}
