package guard

import (
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/walterchris/FirmwareResultServer/pkg/entry"
	"github.com/walterchris/FirmwareResultServer/pkg/event"
)

func remove(s []event.Event, i int) []event.Event {
	s[i] = s[len(s)-1]
	// We do not need to put s[i] at the end, as it will be discarded anyway
	return s[:len(s)-1]
}

func getTimedOutEvents(e []event.Event) []event.Event {

	var r []event.Event

	for _, v := range e {
		if time.Now().Local().After(v.TimeoutTime) {
			r = append(r, v)
		}
	}

	if len(r) == 0 {
		return nil
	}

	return r
}

// Run - Start the Guardian Runner
func Run(pipeline chan event.Event, postPipeline chan event.Event) error {

	log.Infof("Setting up Guardian..")

	var jobs []event.Event

	// Get Jobs from Database
	jobs, err := entry.GetAllOpenEvents()
	if err != nil {
		log.Errorf("Error: %v", err)
	}

	log.Debug("Waiting for Input now.")
	for {
		select {
		case job := <-pipeline:
			switch job.Task {
			case event.CMD_WATCH:
				jobs = append(jobs, job)
				log.Infof("Received new Job %s from Worker #%s we need to track.", job.Hash, job.WorkerID)
				break
			case event.CMD_JOB_END:
				log.Infof("Job %s has ended", job.Hash)
				for i, v := range jobs {
					if v.ProjectID == job.ProjectID &&
						v.WorkerID == job.WorkerID &&
						v.Hash == job.Hash {
						postPipeline <- job
						// Send event to Bruce
						jobs = remove(jobs, i)
					}
				}
				break
			}
		case <-time.After(10 * time.Second):
			timeoutEvents := getTimedOutEvents(jobs)
			if timeoutEvents != nil {
				log.Infof("Timed-out Events: %v", timeoutEvents)
				for _, w := range timeoutEvents {
					for i, v := range jobs {
						if v.ProjectID == w.ProjectID &&
							v.WorkerID == w.WorkerID &&
							v.Hash == w.Hash {
							// Update Entry
							e := entry.Result{
								WorkerID:    w.WorkerID,
								ProjectID:   w.ProjectID,
								Hash:        w.Hash,
								Success:     entry.SUCCESS_FAILURE,
								ReportLink:  "",
								Description: v.Description,
							}
							entry.UpdateWithoutPipeline(&e)
							// Send event to Bruce
							v.Success = entry.SUCCESS_FAILURE
							postPipeline <- v
							jobs = remove(jobs, i)
							break
						}
					}
				}
			}
			log.Debugf("Events: %v", jobs)
		}
	}

	return nil
}
