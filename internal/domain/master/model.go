package master

import "time"

// --- Generic master item (untuk 12 resource) ---

type MasterItem struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	SortOrder int       `json:"sortOrder"`
	IsActive  bool      `json:"isActive"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// --- Bentuk khusus per resource ---

type Subdirectorate struct {
	MasterItem
	DirectorateID string `json:"directorateId"`
	Directorate   string `json:"directorate"`
}

type DocumentType struct {
	ID          string    `json:"id"`
	Code        string    `json:"code"`
	Description string    `json:"description"`
	SortOrder   int       `json:"sortOrder"`
	IsActive    bool      `json:"isActive"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type TestType struct {
	ID               string    `json:"id"`
	Code             string    `json:"code"`
	RequiresPrevious bool      `json:"requiresPrevious"`
	AllowParallel    bool      `json:"allowParallel"`
	SortOrder        int       `json:"sortOrder"`
	IsActive         bool      `json:"isActive"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
}

type ProjectRole struct {
	MasterItem
	Code     string `json:"code"`
	IsSystem bool   `json:"isSystem"`
}

type DependencySystem struct {
	MasterItem
	IsOther bool `json:"isOther"`
}

// --- Employee ---

type Employee struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	DirectorateID   string    `json:"directorateId"`
	Directorate     string    `json:"directorate"`
	SubdirectorateID string   `json:"subdirectorateId"`
	Subdirectorate  string    `json:"subdirectorate"`
	Roles           []string  `json:"roles"`
	RoleIDs         []string  `json:"roleIds"`
	UserID          *string   `json:"userId"`
	IsActive        bool      `json:"isActive"`
}

// --- Request bodies ---

type CreateMasterRequest struct {
	Name             string  `json:"name"`
	Code             string  `json:"code"`
	Description      string  `json:"description"`
	DirectorateID    string  `json:"directorateId"`
	RequiresPrevious *bool   `json:"requiresPrevious"`
	AllowParallel    *bool   `json:"allowParallel"`
}

type UpdateMasterRequest struct {
	Name      *string `json:"name"`
	IsActive  *bool   `json:"isActive"`
	SortOrder *int    `json:"sortOrder"`
}

type CreateEmployeeRequest struct {
	Name             string   `json:"name" validate:"required"`
	DirectorateID    string   `json:"directorateId" validate:"required,uuid"`
	SubdirectorateID string   `json:"subdirectorateId" validate:"required,uuid"`
	RoleIDs          []string `json:"roleIds" validate:"required,min=1"`
}

type UpdateEmployeeRequest struct {
	Name             *string  `json:"name"`
	DirectorateID    *string  `json:"directorateId"`
	SubdirectorateID *string  `json:"subdirectorateId"`
	RoleIDs          []string `json:"roleIds"`
	IsActive         *bool    `json:"isActive"`
}

// --- Lookups response ---

type LookupsResponse struct {
	Directorates      []MasterItem       `json:"directorates"`
	Subdirectorates   []Subdirectorate   `json:"subdirectorates"`
	Vendors           []MasterItem       `json:"vendors"`
	ProjectCategories []MasterItem       `json:"projectCategories"`
	ProjectPriorities []MasterItem       `json:"projectPriorities"`
	ApprovalLevels    []MasterItem       `json:"approvalLevels"`
	ProjectRoles      []ProjectRole      `json:"projectRoles"`
	TaskCategories    []MasterItem       `json:"taskCategories"`
	MeetingCategories []MasterItem       `json:"meetingCategories"`
	DependencySystems []DependencySystem `json:"dependencySystems"`
	DocumentTypes     []DocumentType     `json:"documentTypes"`
	TestTypes         []TestType         `json:"testTypes"`
}

// --- IN_USE error detail ---

type InUseReference struct {
	Entity string `json:"entity"`
	Count  int    `json:"count"`
}

type InUseDetails struct {
	References []InUseReference `json:"references"`
	Suggestion string           `json:"suggestion"`
}
