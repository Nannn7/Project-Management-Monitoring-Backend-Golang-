package auth

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
	"github.com/yourorg/pmo-agi-be/internal/middleware"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}
	if err := h.validate.Struct(req); err != nil {
		response.ValidationError(w, validationFields(err))
		return
	}

	result, err := h.svc.Login(r.Context(), req)
	if err != nil {
		response.Error(w, http.StatusUnauthorized, "UNAUTHENTICATED", err.Error())
		return
	}
	response.Success(w, result)
}

// POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var req RefreshRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}

	result, err := h.svc.RefreshToken(r.Context(), req.RefreshToken)
	if err != nil {
		response.Unauthorized(w)
		return
	}
	response.Success(w, result)
}

// POST /auth/logout
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	// Stateless JWT: client hapus token dari storage
	// Implementasi blacklist token bisa ditambahkan nanti jika perlu
	response.NoContent(w)
}

// GET /auth/me
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	if user == nil {
		response.Unauthorized(w)
		return
	}

	info, err := h.svc.Me(r.Context(), user.UserID)
	if err != nil {
		response.NotFound(w)
		return
	}
	response.Success(w, info)
}

// --- Helper: map validator errors ke fields map ---
func validationFields(err error) map[string]string {
	fields := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			fields[fe.Field()] = fe.Tag()
		}
	}
	return fields
}
