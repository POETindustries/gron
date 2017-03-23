/*******************************************************************
 * Copyright (c) 2017 POET Industries
 *
 * This code is distributed under the MIT License. For a
 * complete list of terms see accompanying LICENSE file
 * or copy at http://opensource.org/licenses/MIT
 *******************************************************************/

package main

import (
	"encoding/json"
	"gron/job"
	"io/ioutil"
	"os"
	"sync"
)

func main() {
	os.RemoveAll("log")
	os.Mkdir("log", 0700)

	jobs := []*job.Job{}
	if data, err := ioutil.ReadFile("jobs.json"); err != nil {
		println("Error reading job file:", err.Error())
	} else if err = json.Unmarshal(data, &jobs); err != nil {
		println("Error parsing job data:", err.Error())
	}

	for i, j := range jobs {
		j.ID = i + 1
	}

	wg := &sync.WaitGroup{}
	wg.Add(len(jobs))

	for _, j := range jobs {
		go j.Start(wg)
	}
	wg.Wait()
}
