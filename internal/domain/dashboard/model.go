package dashboard

import "time"

type DashboardResponse struct {
	Summary            Summary            `json:"summary"`
	StatusDistribution []StatusCount      `json:"statusDistribution"`
	ProgressPerProject []ProjectProgress  `json:"progressPerProject"`
	Projects           []ProjectRow       `json:"projects"`
	UpcomingMeetings   []MeetingRow       `json:"upcomingMeetings"`
}

type Summary struct {
	TotalProjects     int `json:"totalProjects"`
	ActiveProjects    int `json:"activeProjects"`
	HoldProjects      int `json:"holdProjects"`
	CancelledProjects int `json:"cancelledProjects"`
	AverageProgress   int `json:"averageProgress"`
}

type StatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type ProjectProgress struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Progress int    `json:"progress"`
}

type ProjectRow struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Directorate    string  `json:"directorate"`
	Subdirectorate string  `json:"subdirectorate"`
	Status         string  `json:"status"`
	CurrentPhase   *PhaseRef `json:"currentPhase"`
	OverallProgress int    `json:"overallProgress"`
}

type PhaseRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stage string `json:"stage"`
}

type MeetingRow struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"projectId"`
	ProjectName string    `json:"projectName"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Date        time.Time `json:"date"`
}
