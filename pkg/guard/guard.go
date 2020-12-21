package guard

import (
	"time"

	"github.com/prometheus/common/log"
	"github.com/walterchris/FirmwareResultServer/pkg/entry"
)

func Run(pipeline chan entry.Job) error {

	log.Infof("Setting up Guardian..")

	var jobs []entry.Job

	for {
		select {
		case job := <-pipeline:
			jobs = append(jobs, job)
			log.Infof("Received new Job we need to track.")
		case <-time.After(30 * time.Second):
			// Handle Jobs
			log.Infof("Timeout")
		}
	}

	return nil
}
