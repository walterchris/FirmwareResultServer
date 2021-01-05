package gerrit

import (
	"fmt"
	"log"
	"strconv"

	"github.com/andygrunwald/go-gerrit"
	"github.com/walterchris/FirmwareResultServer/pkg/entry"
	"github.com/walterchris/FirmwareResultServer/pkg/event"
)

//Gerrit is an object to interface with the Gerrit API
type Gerrit struct {
	InstanceOrHostname string
	Username           string
	Password           string
}

//CreateGerrit creates a new object to interface Gerrit
func CreateGerrit(InstanceOrHostname string, Username string, Password string) (*Gerrit, error) {
	g := Gerrit{
		InstanceOrHostname: InstanceOrHostname,
		Username:           Username,
		Password:           Password,
	}
	return &g, nil
}

//ReportAsString generates one human readable line for the report passed as argument
func (g Gerrit) ReportAsString(e event.Event) (string, error) {

	var status string

	if e.Success == entry.SUCCESS_OK {
		status = "SUCCESS"
	} else if e.Success == entry.SUCCESS_FAILURE {
		status = "FAIL"
	}

	msg := fmt.Sprintf("%s: %s - %s\n", e.Description, status, e.ReportLink)

	return msg, nil
}

//GenerateReport generates the reports for gerrit
func (g Gerrit) GenerateReport(reports []event.Event) (string, error) {
	var msg string

	pass := 0
	fail := 0
	total := 0

	for _, r := range reports {
		switch r.Success {
		case entry.SUCCESS_OK:
			pass++
			break
		case entry.SUCCESS_FAILURE:
			fail++
			break
		default:
			total++
			break
		}

		reportMsg, err := g.ReportAsString(r)
		if err != nil {
			return "", fmt.Errorf("Failed to transform an Event into a String")
		}
		msg += reportMsg
	}

	msg = fmt.Sprintf("Automatic boot test returned (PASS/FAIL/TOTAL): %d/%d/%d\n\n", pass, fail, total+pass+fail) + msg

	msg += "\nPlease note: This test is under development and might not be accurate at all!\n"

	return msg, nil
}

//ReportStatus posts to a message to a gerrit changeid with the 'autogenerated' flag set
func (g Gerrit) ReportStatus(reports []event.Event,
	CommitHash string,
	Score int,
	Message string) error {

	client, err := gerrit.NewClient(g.InstanceOrHostname, nil)
	if err != nil {
		return err
	}
	//client.Authentication.SetBasicAuth(g.Username, g.Password)

	// _, _, err = client.Accounts.GetAccount("self")
	// if err != nil {
	// 	return err
	// }

	opt := &gerrit.QueryChangeOptions{}
	opt.Query = []string{CommitHash}
	opt.AdditionalFields = []string{"CURRENT_REVISION", "CURRENT_COMMIT", "REVIEWER_UPDATES", "REVIEWED"}

	changes, _, err := client.Changes.QueryChanges(opt)
	if err != nil {
		return err
	}

	m, err := g.GenerateReport(reports)
	if err != nil {
		return err
	}
	Message = m

	if len(*changes) > 1 {
		return fmt.Errorf("CommitHash %s returns too many changes", CommitHash)
	}

	if len(*changes) == 0 {
		return fmt.Errorf("Can not find Change with CommitHash %s", CommitHash)
	}

	for _, c := range *changes {
		log.Printf("Project: '%s' -> subject '%s' -> %s%d\n", c.Project,
			c.Subject,
			g.InstanceOrHostname,
			c.Number)
		log.Printf("URL: %s\n", c.URL)
		log.Printf("CurrentRevision: %s\n", c.CurrentRevision)
		log.Printf("Message was '%s'\n", Message)

		return nil

		notifyUsers := ""
		// Update
		if Score > 0 {
			//FIXME: score
			Score = 0
			notifyUsers = "NONE"
		} else {
			//FIXME: score
			Score = 0
			notifyUsers = "ALL"
		}

		// revision can be "CommitHash", "current"
		var input = gerrit.ReviewInput{
			Message:               Message,
			Tag:                   "autogenerated:LAVABootTest",                       // autogenerated: should be used by robots
			Labels:                map[string]string{"Verified": strconv.Itoa(Score)}, // Code-Review or Verified
			Comments:              nil,
			StrictLabels:          false,
			Drafts:                "",
			Notify:                notifyUsers,
			OmitDuplicateComments: true,
			OnBehalfOf:            "",
		}
		fullChangeID := fmt.Sprintf("%s~%s~%s", c.Project, c.Branch, c.ChangeID)
		_, _, err = client.Changes.SetReview(fullChangeID, CommitHash, &input)
		if err != nil {
			return err
		}
	}

	return nil
}
