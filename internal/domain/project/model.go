package project

import "time"

// --- Project list item ---

type ProjectListItem struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	CRNumber         *string     `json:"crNumber"`
	BRNumber         *string     `json:"brNumber"`
	Directorate      string      `json:"directorate"`
	DirectorateID    string      `json:"directorateId"`
	Subdirectorate   string      `json:"subdirectorate"`
	SubdirectorateID string      `json:"subdirectorateId"`
	Category         string      `json:"category"`
	CategoryID       string      `json:"categoryId"`
	Priority         string      `json:"priority"`
	PriorityID       string      `json:"priorityId"`
	Status           string      `json:"status"`
	Vendor           *string     `json:"vendor"`
	VendorID         *string     `json:"vendorId"`
	OverallProgress  int         `json:"overallProgress"`
	CurrentPhase     *PhaseRef   `json:"currentPhase"`
	PMO              []MemberRef `json:"pmo"`
	Developers       []MemberRef `json:"developers"`
	Users            []MemberRef `json:"users"`
}

type PhaseRef struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Stage string `json:"stage"`
}

type MemberRef struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	IsVendor bool   `json:"isVendor,omitempty"`
}

type Member struct {
	ID               string  `json:"id"`
	EmployeeID       *string `json:"employeeId"`
	Name             string  `json:"name"`
	Role             string  `json:"role"`
	IsPrimary        bool    `json:"isPrimary"`
	IsVendor         bool    `json:"isVendor"`
	Directorate      *string `json:"directorate"`
	DirectorateID    *string `json:"directorateId"`
	Subdirectorate   *string `json:"subdirectorate"`
	SubdirectorateID *string `json:"subdirectorateId"`
}

// --- Project detail ---

type ProjectDetail struct {
	ID               string       `json:"id"`
	Name             string       `json:"name"`
	CRNumber         *string      `json:"crNumber"`
	BRNumber         *string      `json:"brNumber"`
	Directorate      string       `json:"directorate"`
	DirectorateID    string       `json:"directorateId"`
	Subdirectorate   string       `json:"subdirectorate"`
	SubdirectorateID string       `json:"subdirectorateId"`
	Category         string       `json:"category"`
	CategoryID       string       `json:"categoryId"`
	Priority         string       `json:"priority"`
	PriorityID       string       `json:"priorityId"`
	Status           string       `json:"status"`
	Vendor           *string      `json:"vendor"`
	VendorID         *string      `json:"vendorId"`
	PONumber         *string      `json:"poNumber"`
	POStatus         string       `json:"poStatus"`
	GitRepo          string       `json:"gitRepo"`
	HasMigration     bool         `json:"hasMigration"`
	OverallProgress  int          `json:"overallProgress"`
	CreatedBy        UserRef      `json:"createdBy"`
	CreatedAt        time.Time    `json:"createdAt"`
	UpdatedAt        time.Time    `json:"updatedAt"`
	Permissions      ProjectPerms `json:"permissions"`
	PMO              []Member     `json:"pmo"`
	Developers       []Member     `json:"developers"`
	Users            []Member     `json:"users"`
	HoldHistory      []HoldRecord `json:"holdHistory"`
	Phases           []Phase      `json:"phases"`
}

type UserRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ProjectPerms struct {
	CanEdit   bool `json:"canEdit"`
	CanDelete bool `json:"canDelete"`
}

type HoldRecord struct {
	ID        string  `json:"id"`
	HeldAt    string  `json:"heldAt"`
	Reason    string  `json:"reason"`
	HeldBy    string  `json:"heldBy"`
	ResumedAt *string `json:"resumedAt"`
	ResumedBy *string `json:"resumedBy"`
}

// --- Phase & sub-resources ---

type Phase struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Stage               string        `json:"stage"`
	SortOrder           int           `json:"sortOrder"`
	StartDate           string        `json:"startDate"`
	TargetLiveDate      string        `json:"targetLiveDate"`
	PreviousPhaseID     *string       `json:"previousPhaseId"`
	Scope               []string      `json:"scope"`
	Note                *string       `json:"note"`
	ProgressPercent     int           `json:"progressPercent"`
	DocumentProgress    int           `json:"documentProgress"`
	DevelopmentProgress int           `json:"developmentProgress"`
	TestingProgress     int           `json:"testingProgress"`
	Documents           []Document    `json:"documents"`
	TestStages          []TestStage   `json:"testStages"`
	Environments        []Environment `json:"environments"`
	Migrations          []Migration   `json:"migrations"`
	Dependencies        []Dependency  `json:"dependencies"`
}

type Document struct {
	ID            string     `json:"id"`
	Type          string     `json:"type"`
	TypeID        string     `json:"typeId"`
	Status        string     `json:"status"`
	ApprovalLevel string     `json:"approvalLevel"`
	SourceType    string     `json:"sourceType"`
	URL           *string    `json:"url"`
	FileID        *string    `json:"fileId"`
	FileName      *string    `json:"fileName"`
	Approvers     []Approver `json:"approvers"`
}

type Approver struct {
	ID         string  `json:"id"`
	EmployeeID string  `json:"employeeId"`
	Name       string  `json:"name"`
	Level      string  `json:"level"`
	LevelID    string  `json:"levelId"`
	Status     string  `json:"status"`
	Date       *string `json:"date"`
}

type TestStage struct {
	ID              string `json:"id"`
	Type            string `json:"type"`
	TypeID          string `json:"typeId"`
	Status          string `json:"status"`
	ProgressPercent int    `json:"progressPercent"`
	PIC             string `json:"pic"`
}

type Environment struct {
	ID            string      `json:"id"`
	Category      string      `json:"category"`
	Label         string      `json:"label"`
	Value         string      `json:"value"`
	Port          *string     `json:"port,omitempty"`
	DNS           *string     `json:"dns,omitempty"`
	Segment       *string     `json:"segment,omitempty"`
	IsTemplate    *bool       `json:"isTemplate,omitempty"`
	MainBranch    *string     `json:"mainBranch,omitempty"`
	DBName        *string     `json:"dbName,omitempty"`
	ContainerName *string     `json:"containerName,omitempty"`
	Note          *string     `json:"note,omitempty"`
	CPU           *string     `json:"cpu,omitempty"`
	Memory        *string     `json:"memory,omitempty"`
	Storage       *string     `json:"storage,omitempty"`
	ExtraSpecs    []ExtraSpec `json:"extraSpecs"`
}

type ExtraSpec struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Value string `json:"value"`
}

type Migration struct {
	ID           string   `json:"id"`
	SourceSystem string   `json:"sourceSystem"`
	DBName       *string  `json:"dbName"`
	TableName    *string  `json:"tableName"`
	Description  string   `json:"description"`
	Status       string   `json:"status"`
	ReportStatus string   `json:"reportStatus"`
	ReportURL    *string  `json:"reportUrl"`
	PIC          string   `json:"pic"`
	Team         []string `json:"team"`
	StartDate    *string  `json:"startDate"`
	EndDate      *string  `json:"endDate"`
}

type Dependency struct {
	ID               string     `json:"id"`
	System           string     `json:"system"`
	SystemID         string     `json:"systemId"`
	CustomSystemName *string    `json:"customSystemName"`
	ServiceName      *string    `json:"serviceName"`
	Description      string     `json:"description"`
	Endpoints        []Endpoint `json:"endpoints"`
}

type Endpoint struct {
	ID    string  `json:"id"`
	Label string  `json:"label"`
	Value string  `json:"value"`
	Note  *string `json:"note"`
}

// --- Request bodies ---

type CreateProjectRequest struct {
	Name             string  `json:"name" validate:"required,max=200"`
	DirectorateID    string  `json:"directorateId" validate:"required,uuid"`
	SubdirectorateID string  `json:"subdirectorateId" validate:"required,uuid"`
	CategoryID       string  `json:"categoryId" validate:"required,uuid"`
	PriorityID       string  `json:"priorityId" validate:"required,uuid"`
	VendorID         *string `json:"vendorId" validate:"omitempty,uuid"`
}

type UpdateProjectRequest struct {
	Name             *string `json:"name"`
	CRNumber         *string `json:"crNumber"`
	BRNumber         *string `json:"brNumber"`
	DirectorateID    *string `json:"directorateId"`
	SubdirectorateID *string `json:"subdirectorateId"`
	CategoryID       *string `json:"categoryId"`
	PriorityID       *string `json:"priorityId"`
	VendorID         *string `json:"vendorId"`
	PONumber         *string `json:"poNumber"`
	POStatus         *string `json:"poStatus"`
	GitRepo          *string `json:"gitRepo"`
	HasMigration     *bool   `json:"hasMigration"`
	Status           *string `json:"status"` // akan ditolak dengan 422
}

type StatusChangeRequest struct {
	Status string  `json:"status" validate:"required,oneof=Active Hold Cancelled"`
	Reason *string `json:"reason"`
}
