/*******************************************************************
 * Copyright (c) 2017 POET Industries
 *
 * This code is distributed under the MIT License. For a
 * complete list of terms see accompanying LICENSE file
 * or copy at http://opensource.org/licenses/MIT
 *******************************************************************/

package job

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"
)

type Status struct {
	OK   bool
	Data []byte
}

type Job struct {
	ID       int    `json:"-"`
	URL      string `json:"url"`
	Date     string `json:"date"`
	Interval string `json:"interval"`
}

func New() *Job {
	return &Job{}
}

func (j *Job) Run(s chan<- Status) {
	res, err := http.Get(j.URL)
	if err != nil {
		s <- Status{Data: []byte(err.Error())}
	} else if data, err := ioutil.ReadAll(res.Body); err != nil {
		s <- Status{Data: []byte(err.Error())}
	} else if res.StatusCode != 200 {
		s <- Status{OK: false, Data: []byte(res.Status)}
	} else {
		s <- Status{OK: true, Data: data}
	}
	res.Body.Close()
}

func (j *Job) Start(wg *sync.WaitGroup) {
	defer wg.Done()
	dur, err := time.ParseDuration(j.Interval)
	if err != nil {
		println("Job.Start:", err.Error())
		return
	}
	status := make(chan Status)
	time.Sleep(j.firstRunInterval())
	ticker := time.NewTicker(dur)
	defer ticker.Stop()
	for range ticker.C {
		go j.Run(status)
		if s := <-status; !s.OK {
			println(j.ID, "failure", string(s.Data))
		} else {
			println(j.ID, "success", string(s.Data))
		}
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
