package main

import (
	"encoding/json"
	"gron/job"
	"io/ioutil"
	"sync"
)

func main() {
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
