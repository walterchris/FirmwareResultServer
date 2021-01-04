package entry

import (
	"database/sql"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/walterchris/FirmwareResultServer/pkg/database"
	"github.com/walterchris/FirmwareResultServer/pkg/event"
)

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

const (
	SUCCESS_ONGOING = 0
	SUCCESS_OK      = 1
	SUCCESS_FAILURE = 2
)

var connection *sql.DB
var pipeline chan event.Event

const createdFormat = "2006-01-02 15:04:05"

// Ping - Ping database
func Ping(e *Job) error {
	return connection.Ping()
}

// Add new Entry to Database
func Add(e *Job) error {
	if connection == nil {
		if err := Init(pipeline); err != nil {
			return err
		}
	}

	log.Infof("Adding %v to the Database", e)
	// Check if Database is still connected
	if err := Ping(e); err != nil {
		if err := database.Reconnect(connection); err != nil {
			return fmt.Errorf("Can not reconnect to Database: %w", err)
		}
	}

	var doubleJob Job
	// Check if entry already exists in DB
	err := connection.QueryRow("SELECT workerID, projectID, commitHash FROM entries WHERE workerID = ? AND projectID = ? AND commitHash = ?", e.WorkerID, e.ProjectID, e.Hash).Scan(&doubleJob.WorkerID, &doubleJob.ProjectID, &doubleJob.Hash)
	if err == nil && e.WorkerID == doubleJob.WorkerID {
		return fmt.Errorf("Entry with Hash %s already exists in DB (Worker: %s, Project: %s)", e.Hash, e.WorkerID, e.ProjectID)
	}

	timeoutTime := time.Now().Local().Add(time.Minute * time.Duration(e.Timeout))

	// Add Entry into the entries list
	_, err = connection.Exec("INSERT INTO entries(workerID, projectID, commitHash, started, timeout, reportLink) VALUES(?, ?, ?, ?, ?, ?)", e.WorkerID, e.ProjectID, e.Hash, time.Now().Format(createdFormat), timeoutTime.Format(createdFormat), "")
	if err != nil {
		return fmt.Errorf("Failed to insert entry into DB with %w", err)
	}

	// Fire into Channel
	pipelineEntry := event.Event{
		Task:        event.CMD_WATCH,
		WorkerID:    e.WorkerID,
		ProjectID:   e.ProjectID,
		Hash:        e.Hash,
		Timeout:     e.Timeout,
		TimeoutTime: timeoutTime,
	}
	pipeline <- pipelineEntry

	return nil
}

// Update existing entry in DB
func Update(e *Result) error {

	err := UpdateWithoutPipeline(e)
	if err != nil {
		return err
	}
	// Fire into Channel
	pipelineEntry := event.Event{
		Task:       event.CMD_JOB_END,
		WorkerID:   e.WorkerID,
		ProjectID:  e.ProjectID,
		Hash:       e.Hash,
		Success:    e.Success,
		ReportLink: e.ReportLink,
	}
	pipeline <- pipelineEntry

	return nil
}

// UpdateWithoutPipeline - Updates the Entry in the DB without adding it to the pipeline
func UpdateWithoutPipeline(e *Result) error {
	if connection == nil {
		if err := Init(pipeline); err != nil {
			return err
		}
	}

	log.Infof("Updating entry in database with %v", e)

	_, err := connection.Exec("UPDATE entries SET finished = ?, status = ?, reportLink = ? WHERE projectID = ? AND workerID = ? AND commitHash = ? AND status = 0 LIMIT 1", time.Now().Format(createdFormat), e.Success, e.ReportLink, e.ProjectID, e.WorkerID, e.Hash)
	if err != nil {
		return fmt.Errorf("Failed to update entry into DB with %w", err)
	}

	return nil
}

// Init - Set everything up
func Init(p chan event.Event) error {
	var err error
	connection, err = database.Init()

	if err != nil {
		return err
	}
	if p != nil {
		pipeline = p
	}

	return nil
}

// GetAllOpenEvents - Get all open Events from the Database
func GetAllOpenEvents() ([]event.Event, error) {

	var events []event.Event

	if connection == nil {
		if err := Init(pipeline); err != nil {
			return nil, err
		}
	}

	// Grab all open events form the DB
	results, err := connection.Query("SELECT workerID, projectID, commitHash, timeout FROM entries WHERE status = 0")
	if err != nil {
		return nil, err
	}

	for results.Next() {
		var job event.Event
		// for each row, scan the result into our tag composite object
		err = results.Scan(&job.WorkerID, &job.ProjectID, &job.Hash, &job.TimeoutTime)
		if err != nil {
			return nil, err
		}

		events = append(events, job)
		// and then print out the tag's Name attribute
		log.Debugf("Fetching old Job from the Database: %v", job)
	}

	return events, nil
}

// InsertGerritIntoDB - Insert the Gerrit Post into the DB
func InsertGerritIntoDB(e *event.Event) error {

	if connection == nil {
		if err := Init(pipeline); err != nil {
			return err
		}
	}

	// Insert into DB
	_, err := connection.Query("INSERT INTO gerritEntries (ProjectID, Hash) VALUES (?, ?)", e.ProjectID, e.Hash)
	if err != nil {
		return err
	}

	return nil
}

// GetAllEventsWithHash - Return all events with the corresponding hash
func GetAllEventsWithHash(e *event.Event) ([]event.Event, error) {

	var resultEvents []event.Event

	if connection == nil {
		if err := Init(pipeline); err != nil {
			return nil, err
		}
	}

	// Grab all open events form the DB
	results, err := connection.Query("SELECT projectID, commitHash, status, reportLink FROM entries WHERE projectID = ? AND commitHash = ?", e.ProjectID, e.Hash)
	if err != nil {
		return nil, err
	}

	for results.Next() {
		var job event.Event
		// for each row, scan the result into our tag composite object
		err = results.Scan(&job.ProjectID, &job.Hash, &job.Success, &job.ReportLink)
		if err != nil {
			return nil, err
		}

		resultEvents = append(resultEvents, job)

		log.Debugf("Result Events: %v", resultEvents)
	}

	// Check if all results are finished
	for _, result := range resultEvents {
		if result.Success == SUCCESS_ONGOING {
			log.Debugf("Not all results are ready yet.")
			return nil, nil
		}
	}

	return resultEvents, nil
}
