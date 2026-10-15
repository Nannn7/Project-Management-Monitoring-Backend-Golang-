package master

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type Handler struct {
	svc *Service
}

// body JSON -> map, dibatasi 1MB
func decodeBody(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	var body map[string]json.RawMessage
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&body); err != nil || body == nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return nil, false
	}
	return body, true
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /lookups
func (h *Handler) GetLookups(w http.ResponseWriter, r *http.Request) {
	data, err := h.svc.GetLookups(r.Context())
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Success(w, data)
}

// GET /master/{resource}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	resource := chi.URLParam(r, "resource")
	q := r.URL.Query()

	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	var isActive *bool
	if v := q.Get("isActive"); v != "" {
		b := v == "true"
		isActive = &b
	}

	items, total, err := h.svc.ListResource(r.Context(), resource, q.Get("q"), q.Get("sort"), page, pageSize, isActive)
	if err != nil {
		response.FromError(w, r, err)
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
		response.FromError(w, r, err)
		return
	}
	response.List(w, employees, page, pageSize, total)
}

// POST /master/{resource} — stub, implementasi berikutnya
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	item, err := h.svc.CreateResource(r.Context(), chi.URLParam(r, "resource"), body)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Created(w, item)
}

// PATCH /master/{resource}/{id} — stub
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	body, ok := decodeBody(w, r)
	if !ok {
		return
	}
	item, err := h.svc.UpdateResource(r.Context(), chi.URLParam(r, "resource"), chi.URLParam(r, "id"), body)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Success(w, item)
}

// DELETE /master/{resource}/{id} — stub
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	if err := h.svc.DeleteResource(r.Context(), chi.URLParam(r, "resource"), chi.URLParam(r, "id")); err != nil {
		response.FromError(w, r, err)
		return
	}
	response.NoContent(w)
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
