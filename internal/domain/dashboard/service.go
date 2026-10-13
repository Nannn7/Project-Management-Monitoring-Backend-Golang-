package dashboard

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

// GetDashboard — GET /dashboard
// Hanya menghitung project yang boleh diakses pengguna (scope sama dengan project list)
func (s *Service) GetDashboard(ctx context.Context, userID string, projectLimit, meetingLimit int) (*DashboardResponse, error) {
	resp := &DashboardResponse{
		StatusDistribution: []StatusCount{
			{Status: "Active", Count: 0},
			{Status: "Hold", Count: 0},
			{Status: "Cancelled", Count: 0},
		},
		ProgressPerProject: []ProjectProgress{},
		Projects:           []ProjectRow{},
		UpcomingMeetings:   []MeetingRow{},
	}

	// TODO: implementasi query setelah tabel project terisi data
	// Query inti dari spec §4:
	//
	// -- summary
	// SELECT count(*),
	//        count(*) filter (where status = 'Active')    as active,
	//        count(*) filter (where status = 'Hold')      as hold,
	//        count(*) filter (where status = 'Cancelled') as cancelled,
	//        coalesce(round(avg(overall_progress)), 0)    as avg_progress
	// FROM projects p WHERE /* scopePredicate, $1=userID */;
	//
	// -- fase berjalan per project (sort_order terbesar)
	// SELECT DISTINCT ON (project_id) project_id, id, name, stage
	// FROM phases WHERE project_id = ANY($2)
	// ORDER BY project_id, sort_order DESC;
	//
	// -- upcoming meetings
	// SELECT m.id, m.project_id, p.name, m.title, mc.name, m.scheduled_at
	// FROM meetings m
	// JOIN projects p ON p.id = m.project_id
	// JOIN master_meeting_categories mc ON mc.id = m.category_id
	// WHERE m.status = 'Scheduled' AND m.scheduled_at >= now()
	//   AND /* scopePredicate */
	// ORDER BY m.scheduled_at ASC LIMIT $meetingLimit;

	return resp, nil
}
