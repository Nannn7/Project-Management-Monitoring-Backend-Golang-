package project

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
	"github.com/yourorg/pmo-agi-be/internal/middleware"
	"github.com/yourorg/pmo-agi-be/internal/response"
	"github.com/yourorg/pmo-agi-be/internal/validation"
)

type Handler struct {
	svc      *Service
	validate *validator.Validate
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc, validate: validator.New()}
}

// GET /projects
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	q := r.URL.Query().Get("q")
	status := r.URL.Query().Get("status")
	priorityID := r.URL.Query().Get("priorityId")
	categoryID := r.URL.Query().Get("categoryId")
	directorateID := r.URL.Query().Get("directorateId")
	sort := r.URL.Query().Get("sort")
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("pageSize"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	projects, total, err := h.svc.ListProjects(r.Context(), user.UserID, q, status, priorityID, categoryID, directorateID, sort, page, pageSize)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.List(w, projects, page, pageSize, total)
}

// GET /projects/{id}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	project, err := h.svc.GetProject(r.Context(), user.UserID, chi.URLParam(r, "id"))
	if err != nil {
		response.FromError(w, r, err) // 404 untuk "tidak ada" dan "dilarang" datang dari service
		return
	}
	response.Success(w, project)
}

// POST /projects
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	var req CreateProjectRequest
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields() // createdBy, status, dst bukan hak client
	if err := dec.Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}

	// normalisasi sebelum validasi: "  " bukan nama, "" bukan vendor
	req.Name = strings.TrimSpace(req.Name)
	if req.VendorID != nil && strings.TrimSpace(*req.VendorID) == "" {
		req.VendorID = nil
	}
	if fields := validation.Fields(req); fields != nil {
		response.ValidationError(w, fields)
		return
	}

	project, err := h.svc.CreateProject(r.Context(), user.UserID, req)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Created(w, project)
}

// PATCH /projects/{id}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	projectID := chi.URLParam(r, "id")

	var req UpdateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}

	// Tolak field status — sesuai spec §6 PATCH /projects/{id}
	if req.Status != nil {
		response.ValidationError(w, map[string]string{
			"status": "Gunakan POST /projects/{id}/status untuk mengubah status.",
		})
		return
	}

	project, err := h.svc.UpdateProject(r.Context(), user.UserID, projectID, req)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Success(w, project)
}

// DELETE /projects/{id}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	projectID := chi.URLParam(r, "id")

	if err := h.svc.DeleteProject(r.Context(), user.UserID, projectID); err != nil {
		response.FromError(w, r, err)
		return
	}
	response.NoContent(w)
}

// POST /projects/{id}/status
func (h *Handler) ChangeStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	projectID := chi.URLParam(r, "id")

	var req StatusChangeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Body tidak valid.")
		return
	}
	if fields := validation.Fields(req); fields != nil {
		response.ValidationError(w, fields)
		return
	}

	result, err := h.svc.ChangeStatus(r.Context(), user.UserID, projectID, req)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Success(w, result)
}

// --- Stub handlers (akan diimplementasi di Phase 4) ---

func (h *Handler) GetTasks(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: GET /projects/{id}/tasks")
}

func (h *Handler) GetMeetings(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: GET /projects/{id}/meetings")
}

func (h *Handler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: GET /projects/{id}/audit-logs")
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /projects/{id}/members")
}

func (h *Handler) UpdateMember(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /members/{id}")
}

func (h *Handler) DeleteMember(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /members/{id}")
}

func (h *Handler) AddPhase(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /projects/{id}/phases")
}

func (h *Handler) UpdatePhase(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /phases/{id}")
}

func (h *Handler) AddEnvironment(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /phases/{id}/environments")
}

func (h *Handler) UpdateEnvironment(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /environments/{id}")
}

func (h *Handler) DeleteEnvironment(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /environments/{id}")
}

func (h *Handler) AddDocument(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /phases/{id}/documents")
}

func (h *Handler) UpdateDocument(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /documents/{id}")
}

func (h *Handler) DeleteDocument(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /documents/{id}")
}

func (h *Handler) AddApprover(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /documents/{id}/approvers")
}

func (h *Handler) UpdateApprover(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /approvers/{id}")
}

func (h *Handler) DeleteApprover(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /approvers/{id}")
}

func (h *Handler) AddMigration(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /phases/{id}/migrations")
}

func (h *Handler) UpdateMigration(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /migrations/{id}")
}

func (h *Handler) DeleteMigration(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /migrations/{id}")
}

func (h *Handler) AddDependency(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: POST /phases/{id}/dependencies")
}

func (h *Handler) UpdateDependency(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /dependencies/{id}")
}

func (h *Handler) DeleteDependency(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: DELETE /dependencies/{id}")
}

func (h *Handler) ReviewTask(w http.ResponseWriter, r *http.Request) {
	response.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "TODO: PATCH /tasks/{id}/review")
}

// Helper
func validationFields(err error) map[string]string {
	fields := make(map[string]string)
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			fields[fe.Field()] = fe.Tag()
		}
	}
	return fields
}
