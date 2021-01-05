package event

import "time"

const (
	CMD_WATCH    = 0
	CMD_JOB_END  = 1
	CMD_JOB_DONE = 2
)

// Event Type
type Event struct {
	Task        int    // 0 = Watch, 1 = Done, 2 = Failed
	WorkerID    string `json:"workerID"`
	ProjectID   string `json:"projectID"`
	Hash        string `json:"hash"`
	Timeout     int    `json:"timeout"`
	Success     int    `json:"success"`
	ReportLink  string `json:"reportLink"`
	Description string `json:"description"`
	TimeoutTime time.Time
}
