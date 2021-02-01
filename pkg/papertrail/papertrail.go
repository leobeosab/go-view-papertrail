package papertrail

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/muesli/termenv"
)

var (
	papertrailKey string
	client        = &http.Client{}
	metaRe        = regexp.MustCompile("(\\w*?){1}(?: )\\[(.*?)\\]-\\((.*?)\\)\\:(.*?)\\{")
	jsonRe        = regexp.MustCompile("(?:loggedObject\\: )'(.*?)'")
)

const (
	papertrailURL = "https://papertrailapp.com/api/v1/events/search.json"
	logLimit      = 100
)

// PTResponse formatted response from papertrail api
type ptResponse struct {
	Events []ptLog `json:"events"`
}

// PTLog log format of papertrail
type ptLog struct {
	SourceIP string `json:"source_ip"`
	Program  string `json:"program"`
	Message  string `json:"message"`
	Date     string `json:"generated_at"`
	Hostname string `json:"hostname"`
	Severity string `json:"severity"`
}

// Log this is the struct we use for displaying information
type Log struct {
	Env      string
	Program  string
	Severity string
	Label    string
	Message  string
	JSON     string
	Date     string
}

// Display return string to be displayed in console
func (l Log) Display(color bool, term termenv.Profile) string {
	s := ""

	var env string
	var severity string

	if color {
		env = termenv.String(l.Env).Foreground(term.Color("14")).String()
		severity = displaySeverity(l.Severity, term)
	} else {
		severity = " " + l.Severity + " "
		env = l.Env
	}

	dt, _ := time.Parse(time.RFC3339, l.Date)
	date := dt.Format("2006-1-2 15:04")

	s += date + " "
	s += "[" + env + "]"
	s += " - "
	s += severity
	s += " "
	s += "(" + l.Label + ") ~"
	s += l.Message

	return s
}

func displaySeverity(s string, term termenv.Profile) string {
	var background string
	switch strings.ToLower(s) {
	case "error":
		background = "1"
	case "warning":
		background = "11"
	case "info":
		background = "10"
	default:
		background = "15"
	}

	s = " " + s + " "

	return termenv.String(s).Foreground(term.Color("0")).Background(term.Color(background)).String()
}

// Init papertrail
func Init() {
	// TODO:// pull from file or env
	papertrailKey = os.Getenv("PAPERTRAIL_KEY")
}

// GetLogs requests logs from papertrail and format to be our format
func GetLogs(query string) []Log {
	result, success := sendPapertrailRequest(query)
	if !success {
		return []Log{}
	}

	var formattedLogs ptResponse

	err := json.Unmarshal([]byte(result), &formattedLogs)
	if err != nil {
		panic(err)
	}

	var logs []Log

	for _, ptL := range formattedLogs.Events {
		// step one format message
		msg := ptL.Message
		metaMatch := metaRe.FindStringSubmatch(msg)
		if len(metaMatch) < 5 {
			continue
		}

		var json string

		jsonMatch := jsonRe.FindStringSubmatch(msg)
		if jsonMatch == nil || len(jsonMatch) < 2 {
			json = `{"error": "Could not parse JSON"}`
		} else {
			json = jsonMatch[1]
		}

		l := Log{
			Date:     ptL.Date,
			Program:  ptL.Program,
			Severity: metaMatch[1],
			Env:      metaMatch[2],
			Label:    metaMatch[3],
			Message:  metaMatch[4],
			JSON:     json,
		}

		logs = append(logs, l)
	}

	return logs //formattedLogs.Events
}

func sendPapertrailRequest(query string) (string, bool) {
	req, err := http.NewRequest("GET", papertrailURL, nil)
	if err != nil {
		panic(err)
	}

	req.Header.Add("X-Papertrail-Token", papertrailKey)

	q := req.URL.Query()
	q.Add("q", query)
	q.Add("limit", fmt.Sprint(logLimit))
	req.URL.RawQuery = q.Encode()

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		return "", false
	}

	return string(body), true
}
