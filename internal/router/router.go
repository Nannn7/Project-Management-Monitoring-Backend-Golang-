package router

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/yourorg/pmo-agi-be/internal/config"
	"github.com/yourorg/pmo-agi-be/internal/domain/auth"
	"github.com/yourorg/pmo-agi-be/internal/domain/dashboard"
	"github.com/yourorg/pmo-agi-be/internal/domain/master"
	"github.com/yourorg/pmo-agi-be/internal/domain/project"
	"github.com/yourorg/pmo-agi-be/internal/middleware"
)

func New(db *pgxpool.Pool) http.Handler {
	r := chi.NewRouter()

	// --- Global middleware ---
	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.RequestID)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{config.App.CORSOrigin},
		AllowedMethods:   []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// --- Inisialisasi services & handlers ---
	authSvc := auth.NewService(db)
	authH := auth.NewHandler(authSvc)

	masterSvc := master.NewService(db)
	masterH := master.NewHandler(masterSvc)

	projectSvc := project.NewService(db)
	projectH := project.NewHandler(projectSvc)

	dashboardSvc := dashboard.NewService(db)
	dashboardH := dashboard.NewHandler(dashboardSvc)

	// Shorthand permission middleware
	requirePerm := func(menu, action string) func(http.Handler) http.Handler {
		return middleware.RequirePermission(db, menu, action)
	}

	// --- Routes ---
	r.Route("/api/v1", func(r chi.Router) {

		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok"}`))
		})

		// Auth (public)
		r.Route("/auth", func(r chi.Router) {
			r.Post("/login", authH.Login)
			r.Post("/refresh", authH.Refresh)
			r.With(middleware.Auth).Post("/logout", authH.Logout)
			r.With(middleware.Auth).Get("/me", authH.Me)
		})

		// Protected routes (semua butuh JWT)
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth)

			// Lookups (semua pengguna login)
			r.Get("/lookups", masterH.GetLookups)

			// Dashboard
			r.With(requirePerm("dashboard", "view")).Get("/dashboard", dashboardH.Get)

			// Projects
			r.Route("/projects", func(r chi.Router) {
				r.With(requirePerm("projects", "view")).Get("/", projectH.List)
				r.With(requirePerm("projects", "create")).Post("/", projectH.Create)

				r.Route("/{id}", func(r chi.Router) {
					r.With(requirePerm("projects", "view")).Get("/", projectH.Get)
					r.With(requirePerm("projects", "edit")).Patch("/", projectH.Update)
					r.With(requirePerm("projects", "delete")).Delete("/", projectH.Delete)
					r.With(requirePerm("projects", "edit")).Post("/status", projectH.ChangeStatus)

					// Sub-resources project
					r.With(requirePerm("projects", "edit")).Post("/members", projectH.AddMember)
					r.With(requirePerm("projects", "edit")).Post("/phases", projectH.AddPhase)
					r.With(requirePerm("projects", "view")).Get("/tasks", projectH.GetTasks)
					r.With(requirePerm("projects", "view")).Get("/meetings", projectH.GetMeetings)
					r.With(requirePerm("projects", "view")).Get("/audit-logs", projectH.GetAuditLogs)
				})
			})

			// Members (standalone PATCH/DELETE)
			r.With(requirePerm("projects", "edit")).Patch("/members/{id}", projectH.UpdateMember)
			r.With(requirePerm("projects", "edit")).Delete("/members/{id}", projectH.DeleteMember)

			// Phases
			r.With(requirePerm("projects", "edit")).Patch("/phases/{id}", projectH.UpdatePhase)
			r.With(requirePerm("projects", "edit")).Post("/phases/{id}/environments", projectH.AddEnvironment)
			r.With(requirePerm("projects", "edit")).Post("/phases/{id}/documents", projectH.AddDocument)
			r.With(requirePerm("projects", "edit")).Post("/phases/{id}/migrations", projectH.AddMigration)
			r.With(requirePerm("projects", "edit")).Post("/phases/{id}/dependencies", projectH.AddDependency)

			// Environments
			r.With(requirePerm("projects", "edit")).Patch("/environments/{id}", projectH.UpdateEnvironment)
			r.With(requirePerm("projects", "edit")).Delete("/environments/{id}", projectH.DeleteEnvironment)

			// Documents
			r.With(requirePerm("projects", "edit")).Patch("/documents/{id}", projectH.UpdateDocument)
			r.With(requirePerm("projects", "edit")).Delete("/documents/{id}", projectH.DeleteDocument)
			r.With(requirePerm("projects", "edit")).Post("/documents/{id}/approvers", projectH.AddApprover)

			// Approvers
			r.With(requirePerm("projects", "edit")).Patch("/approvers/{id}", projectH.UpdateApprover)
			r.With(requirePerm("projects", "edit")).Delete("/approvers/{id}", projectH.DeleteApprover)

			// Migrations
			r.With(requirePerm("projects", "edit")).Patch("/migrations/{id}", projectH.UpdateMigration)
			r.With(requirePerm("projects", "edit")).Delete("/migrations/{id}", projectH.DeleteMigration)

			// Dependencies
			r.With(requirePerm("projects", "edit")).Patch("/dependencies/{id}", projectH.UpdateDependency)
			r.With(requirePerm("projects", "edit")).Delete("/dependencies/{id}", projectH.DeleteDependency)

			// Tasks review
			r.With(requirePerm("projects", "edit")).Patch("/tasks/{id}/review", projectH.ReviewTask)

			// Master Data
			r.Route("/master", func(r chi.Router) {
				r.With(requirePerm("master-data", "view")).Get("/{resource}", masterH.List)
				r.With(requirePerm("master-data", "create")).Post("/{resource}", masterH.Create)
				r.With(requirePerm("master-data", "edit")).Patch("/{resource}/{id}", masterH.Update)
				r.With(requirePerm("master-data", "delete")).Delete("/{resource}/{id}", masterH.Delete)
			})

			// Employees
			r.With(middleware.RequireAnyPermission(db,
				middleware.MenuAction{Menu: "projects", Action: "view"},
				middleware.MenuAction{Menu: "master-data", Action: "view"},
			)).Get("/employees", masterH.ListEmployees)
			r.With(requirePerm("master-data", "create")).Post("/employees", masterH.CreateEmployee)
			r.With(requirePerm("master-data", "edit")).Patch("/employees/{id}", masterH.UpdateEmployee)
			r.With(requirePerm("master-data", "delete")).Delete("/employees/{id}", masterH.DeleteEmployee)
		})
	})

	return r
}
