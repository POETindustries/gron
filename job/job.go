/*******************************************************************
 * Copyright (c) 2017 POET Industries
 *
 * This code is distributed under the MIT License. For a
 * complete list of terms see accompanying LICENSE file
 * or copy at http://opensource.org/licenses/MIT
 *******************************************************************/

package job

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sail/email"
	"strconv"
	"sync"
	"time"
)

// Status contains information about a job that has just finished running.
type Status struct {
	OK   bool
	Data []byte
}

// Job runs the cron job at the location designated by URL. It handles
// execution scheduling, status logging and e-mail notification in
// case of errors.
type Job struct {
	ID       int    `json:"-"`
	URL      string `json:"url"`
	Date     string `json:"date"`
	Interval string `json:"interval"`

	KeepLog         bool   `json:"notify_log"`
	NotifyByMail    bool   `json:"notify_mail"`
	NotifyOnFailure bool   `json:"notify_on_failure"`
	NotifyInterval  string `json:"notify_interval"`

	MailUser string `json:"mail_user"`
	MailPass string `json:"mail_password"`
	MailHost string `json:"mail_host_smtp"`
	MailPort uint16 `json:"mail_port_smtp"`
}

// New acts as Job constructor and returns an empty Job struct.
func New() *Job {
	return &Job{}
}

// Run executes the cron job and generates a Status object which is
// passed to the receiver channel s.
func (j *Job) Run(s chan<- *Status) {
	stat := &Status{}
	res, err := http.Get(j.URL)
	if err != nil {
		stat.Data = []byte(err.Error())
	} else if data, err := ioutil.ReadAll(res.Body); err != nil {
		stat.Data = []byte(err.Error())
	} else if res.StatusCode != 200 {
		stat.Data = []byte(res.Status)
	} else {
		stat.OK = true
		stat.Data = data
	}
	res.Body.Close()
	s <- stat
}

// Start handles Job initialization such as date and interval parsing
// and starts the necessary tickers as well as the main execution loop.
func (j *Job) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	deltaTick, err := time.ParseDuration(j.Interval)
	if err != nil {
		return
	}
	deltaNotify, err := time.ParseDuration(j.NotifyInterval)
	if err != nil {
		return
	}
	status := make(chan *Status)
	time.Sleep(j.firstRunInterval(deltaTick))
	ticker := time.NewTicker(deltaTick)
	defer ticker.Stop()
	notifier := time.NewTicker(deltaNotify)
	defer notifier.Stop()
	for {
		select {
		case <-ticker.C:
			go j.Run(status)
			j.handleStatus(<-status)
		case <-notifier.C:
			if !j.NotifyByMail || j.notify(nil) {
				os.Remove("log/" + strconv.Itoa(j.ID) + ".log")
			}
		}
	}
}

func (j *Job) firstRunInterval(interval time.Duration) time.Duration {
	start, err := time.Parse("2006-01-02 15:04:05", j.Date)
	if err == nil {
		dur := time.Until(start)
		for dur < 0 {
			dur += interval
		}
		return dur
	}
	return 0
}

func (j *Job) handleStatus(s *Status) {
	if j.KeepLog {
		j.log(s)
	}
	if !s.OK && j.NotifyOnFailure {
		j.notify(s)
	}
}

func (j *Job) log(s *Status) {
	path := "log/" + strconv.Itoa(j.ID) + ".log"
	flag := os.O_WRONLY | os.O_APPEND | os.O_CREATE
	if f, err := os.OpenFile(path, flag, 0600); err == nil {
		defer f.Close()
		line := fmt.Sprintf("%s Job %d %s: Success? %t. Message: %s\r\n",
			time.Now().Format("2006-01-02 15:04:05"), j.ID, j.URL, s.OK, s.Data)
		f.WriteString(line)
	}
}

// Notify sends e-mail notification pertaining to the related job.
// It takes one argument, a Status object s. If s is not nil, an
// e-mail will be sent containing only the information about the
// execution attempt that produced s. If the argument is nil, an
// e-mail is created that contains all status messages currently
// held in the log. Notify does not flush the log by itself.
func (j *Job) notify(s *Status) bool {
	sender := &email.Sender{
		Name:    "Gron Webcron Server",
		Address: j.MailUser,
		Pass:    j.MailPass,
		Host:    j.MailHost,
		Port:    j.MailPort}
	mail := email.New(sender)
	mail.Subject = time.Now().Format("2006-01-02") + ": Gron Status Summary"
	mail.To = []email.Recipient{{Address: j.MailUser}}
	if s != nil {
		mail.Body = fmt.Sprintf("%s Job %d %s: Success? %t. Message: %s\r\n",
			time.Now().Format("2006-01-02 15:04:05"), j.ID, j.URL, s.OK, s.Data)
	} else {
		stats, err := ioutil.ReadFile("log/" + strconv.Itoa(j.ID) + ".log")
		if err != nil {
			return false
		}
		mail.Body = string(stats)
	}
	if err := mail.Send(); err != nil {
		j.log(&Status{Data: []byte(err.Error())})
		return false
	}
	return true
}
