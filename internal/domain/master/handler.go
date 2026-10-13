package master

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /lookups
func (h *Handler) GetLookups(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetLookups(r.Context())
	if err != nil {
		response.InternalError(w)
		return
	}
	response.Success(w, data)
}

// GET /master/{resource}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	if _, ok := ValidateResource(resource); !ok {
		response.NotFound(w)
		return
	}

	q := r.URL.Query().Get("q")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var isActiveFilter *bool
	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		b := isActiveStr == "true"
		isActiveFilter = &b
	}

	items, total, err := h.svc.ListResource(r.Context(), resource, q, page, pageSize, isActiveFilter)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.List(w, items, page, pageSize, total)
}

// GET /employees
func (h *Handler) ListEmployees(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query().Get("q")
	role := r.URL.Query().Get("role")
	directorateID := r.URL.Query().Get("directorateId")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	var isActiveFilter *bool
	if isActiveStr := r.URL.Query().Get("isActive"); isActiveStr != "" {
		b := isActiveStr == "true"
		isActiveFilter = &b
	}

	employees, total, err := h.svc.ListEmployees(r.Context(), q, role, directorateID, isActiveFilter, page, pageSize)
	if err != nil {
		response.InternalError(w)
		return
	}
	response.List(w, employees, page, pageSize, total)
}

// POST /master/{resource} — stub, implementasi berikutnya
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi create master")
}

// PATCH /master/{resource}/{id} — stub
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi update master")
}

// DELETE /master/{resource}/{id} — stub
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi delete master")
}

// POST /employees — stub
func (h *Handler) CreateEmployee(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi create employee")
}

// PATCH /employees/{id} — stub
func (h *Handler) UpdateEmployee(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi update employee")
}

// DELETE /employees/{id} — stub
func (h *Handler) DeleteEmployee(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: implementasi delete employee")
}
