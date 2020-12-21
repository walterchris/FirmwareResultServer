package entry

type Job struct {
	WorkerID  string `json:"workerID"`
	ProjectID string `json:"projectID"`
	Hash      string `json:"hash"`
	Timeout   int    `json:"timeout"`
}

type Result struct {
	WorkerID   string `json:"workerID"`
	ProjectID  string `json:"projectID"`
	Hash       string `json:"hash"`
	Success    int    `json:"success"`
	ReportLink string `json:"reportLink"`
}
