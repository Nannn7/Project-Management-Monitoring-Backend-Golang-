package project

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// scopePredicate — predikat SQL scope project sesuai spec §2.4
// Inject ke semua query yang melibatkan tabel projects
// $1 = userID
const scopePredicate = `
	p.deleted_at IS NULL AND (
	  EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id
	          WHERE ur.user_id = $1 AND ro.name = 'Administrator')
	  OR p.created_by = $1
	  OR EXISTS (
	    SELECT 1 FROM project_members m
	    JOIN employees e ON e.id = m.employee_id
	    JOIN master_project_roles r ON r.id = m.project_role_id
	    WHERE m.project_id = p.id AND e.user_id = $1 AND r.code IN ('PMO','DEVELOPER')
	  )
	)
`

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}
