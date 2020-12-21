package entry

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/prometheus/common/log"
	"github.com/walterchris/FirmwareResultServer/pkg/database"
)

var connection *sql.DB
var pipeline chan Job

const createdFormat = "2006-01-02 15:04:05"

// Ping - Ping database
func (e *Job) Ping() error {
	return connection.Ping()
}

// Update existing entry in DB
func (e *Result) Update() error {
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

// Add new Entry to Database
func (e *Job) Add() error {
	if connection == nil {
		if err := Init(pipeline); err != nil {
			return err
		}
	}

	log.Infof("Adding %v to the Database", e)
	// Check if Database is still connected
	if err := e.Ping(); err != nil {
		if err := database.Reconnect(connection); err != nil {
			return fmt.Errorf("Can not reconnect to Database: %w", err)
		}
	}

	timeoutTime := time.Now().Local().Add(time.Minute * time.Duration(e.Timeout))

	// Add Entry into the entries list
	_, err := connection.Exec("INSERT INTO entries(workerID, projectID, commitHash, started, timeout) VALUES(?, ?, ?, ?, ?)", e.WorkerID, e.ProjectID, e.Hash, time.Now().Format(createdFormat), timeoutTime.Format(createdFormat))
	if err != nil {
		return fmt.Errorf("Failed to insert entry into DB with %w", err)
	}

	// Fire into Channel
	pipelineEntry := *e
	pipeline <- pipelineEntry

	return nil
}

// Init - Set everything up
func Init(p chan Job) error {
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
