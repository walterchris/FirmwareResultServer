package bruce

import (
	"encoding/json"

	log "github.com/sirupsen/logrus"
	"github.com/walterchris/FirmwareResultServer/pkg/entry"
	"github.com/walterchris/FirmwareResultServer/pkg/event"
	"github.com/walterchris/FirmwareResultServer/pkg/gerrit"
)

// Run this Bruce
func Run(p chan event.Event) error {

	log.Infof("Setting up Bruce..")

	for {
		select {
		case event := <-p:
			eventJSON, _ := json.Marshal(event)
			log.Debugf("Incoming Event: %s", string(eventJSON))

			// Check if this event is the last one of this series
			// only one hash from the same project should be there.
			events, err := entry.GetAllEventsWithHash(&event)

			if err != nil {
				log.Errorf("Unable to fetch Events with Hash %s: %v", event.Hash, err)
			}
			if events != nil {
				log.Debugf("Fetched Events for Hash %s: %v", event.Hash, events)
				log.Debugf("Invoking Post To Gerrit now!")
				err = entry.InsertGerritIntoDB(&event)
				if err != nil {
					log.Errorf("Error inserting Gerrit Entry into DB: %v", err)
				}
				g, err := gerrit.CreateGerrit("https://review.coreboot.org/", "hardwaretestrobot", "5heA0O1EZ6aMecPDIiG5Nj7pC6fK+ias3IOtmioplg")
				if err != nil {
					log.Errorf("Error creating gerrit: %v", err)
				}
				err = g.ReportStatus(events, event.Hash, 0, "")
				if err != nil {
					log.Errorf("Error generating reports: %v", err)
				}
			}
		}
	}

	return nil
}
