package dashboard

import (
	"net/http"
	"strconv"

	"github.com/yourorg/pmo-agi-be/internal/middleware"
	"github.com/yourorg/pmo-agi-be/internal/response"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// GET /dashboard
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	projectLimit, _ := strconv.Atoi(r.URL.Query().Get("projectLimit"))
	meetingLimit, _ := strconv.Atoi(r.URL.Query().Get("meetingLimit"))
	if projectLimit < 1 {
		projectLimit = 10
	}
	if meetingLimit < 1 {
		meetingLimit = 5
	}

	data, err := h.svc.GetDashboard(r.Context(), user.UserID, projectLimit, meetingLimit)
	if err != nil {
		response.FromError(w, r, err)
		return
	}
	response.Success(w, data)
}
