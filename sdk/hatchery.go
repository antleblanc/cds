package sdk

// Hatchery registration model
type Hatchery struct {
	ID            int64  `json:"id" db:"id"`
	UID           string `json:"uid" db:"uid"`
	Name          string `json:"name" db:"name"`
	GroupID       int64  `json:"group_id" db:"group_id"`
	WorkerModelID int64  `json:"worker_model_id" db:"worker_model_id"`
	Model         Model  `json:"model" db:"-"`
	Version       string `json:"version" db:"-"`
	Uptodate      bool   `json:"up_to_date" db:"-"`
	IsSharedInfra bool   `json:"is_shared_infra" db:"-"`
	ModelType     string `json:"model_type" db:"model_type"`
	Type          string `json:"type" db:"type"`
	RatioService  int    `json:"ratio_service" db:"ratio_service"`
}
