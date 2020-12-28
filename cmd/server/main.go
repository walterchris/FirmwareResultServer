package main

import (
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/walterchris/FirmwareResultServer/pkg/entry"
	"github.com/walterchris/FirmwareResultServer/pkg/event"
	"github.com/walterchris/FirmwareResultServer/pkg/guard"

	"github.com/gin-gonic/gin"
)

func main() {
	log.SetOutput(os.Stdout)

	// Only log the warning severity or above.
	log.SetLevel(log.DebugLevel)

	router := gin.Default()

	jobPipeline := make(chan event.Event)

	go guard.Run(jobPipeline)
	entry.Init(jobPipeline)

	// Query string parameters are parsed using the existing underlying request object.
	// The request responds to a url matching:  /welcome?firstname=Jane&lastname=Doe
	router.GET("/welcome", func(c *gin.Context) {
		firstname := c.DefaultQuery("firstname", "Guest")
		lastname := c.Query("lastname") // shortcut for c.Request.URL.Query().Get("lastname")

		c.String(http.StatusOK, "Hello %s %s", firstname, lastname)
	})

	/*
	 * workerID
	 * projectID
	 * commitHash
	 * jobDescription => ["PreStep", "Smoke Tests", "Cleanup"]
	 * Timeout => 120 (in Minutes)
	 */
	router.POST("/job/start", func(c *gin.Context) {
		var json entry.Job
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// Add a new Entry
		if err := entry.Add(&json); err != nil {
			log.Errorf("Error while adding entry: %v", err)
		}
	})

	router.POST("/job/end", func(c *gin.Context) {
		var json entry.Result
		if err := c.ShouldBindJSON(&json); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		if err := entry.Update(&json); err != nil {
			log.Errorf("Error while update entry: %v", err)
		}
	})

	router.Run(":8080")
}
