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
	"strconv"
	"sync"
	"time"
)

type Status struct {
	JobID int
	OK    bool
	URL   string
	Data  []byte
}

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

func New() *Job {
	return &Job{}
}

func (j *Job) Run(s chan<- *Status) {
	stat := &Status{JobID: j.ID, URL: j.URL}
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

func (j *Job) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	dur, err := time.ParseDuration(j.Interval)
	if err != nil {
		println("Job.Start:", err.Error())
		return
	}
	status := make(chan *Status)
	time.Sleep(j.firstRunInterval())
	ticker := time.NewTicker(dur)
	defer ticker.Stop()
	for range ticker.C {
		go j.Run(status)
		j.handleStatus(<-status)
	}
}

func (j *Job) firstRunInterval() time.Duration {
	start, err := time.Parse("2006-01-02 15:04:05", j.Date)
	if err == nil {
		if dur := time.Until(start); dur > 0 {
			return dur
		}
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
		line := fmt.Sprintf("%s Job %d %s: Success? %t. %s\r\n",
			time.Now().Format("2006-01-02 15:04:05"), j.ID, j.URL, s.OK, s.Data)
		f.WriteString(line)
	}
}

func (j *Job) notify(s *Status) {
	// TODO 2017-03-23: Implement email notification
}
