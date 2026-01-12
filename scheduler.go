package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type Monitor struct {
	Id          int64
	Url         string
	Interval    time.Duration
	NextRunTime time.Time
}

func check(link string, data chan<- Response) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	var res Response

	var client = &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		res = Response{
			Latency: "0",
			Status:  "down",
		}
		data <- res
		return
	}
	response, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		res = Response{
			Latency: fmt.Sprintf("%v", latency),
			Status:  "down",
		}
		data <- res
		return
	}
	defer response.Body.Close()

	isUp := response.StatusCode >= 200 && response.StatusCode < 400

	if isUp {
		res = Response{
			Latency: fmt.Sprintf("%v", latency),
			Status:  "up",
		}
	} else {
		res = Response{
			Latency: fmt.Sprintf("%v", latency),
			Status:  "down",
		}

	}

	data <- res
}

var (
	ScheduleMonitor []Monitor
	ScheduleMutex   sync.RWMutex
)

func FindNextMonitor() *Monitor {

	ScheduleMutex.RLock()
	defer ScheduleMutex.RUnlock()

	if len(ScheduleMonitor) == 0 {
		return nil
	}

	next := 0
	for i := 1; i < len(ScheduleMonitor); i++ {
		if ScheduleMonitor[i].NextRunTime.Before(ScheduleMonitor[next].NextRunTime) {
			next = i
		}
	}
	return &ScheduleMonitor[next]
}

func CheckMoniter(jobs chan<- *Monitor) {
	for {
		next := FindNextMonitor()
		if next == nil {
			time.Sleep(5 * time.Second)
			continue
		}

		wait := time.Until(next.NextRunTime)
		if wait < 0 {
			wait = 0
		}
		timer := time.NewTimer(wait)
		<-timer.C
		jobs <- next
		ScheduleMutex.Lock()
		next.NextRunTime = time.Now().Add(next.Interval)
		ScheduleMutex.Unlock()
	}
}

func CheckerPool(count int, jobs <-chan *Monitor) {
	for i := 0; i < count; i++ {
		go func(workerID int) {
			for monitor := range jobs {
				resChan := make(chan Response, 1)
				check(monitor.Url, resChan)
				res := <-resChan
				LogCheck(monitor.Id, res)
			}
		}(i)
	}
}
func AddNewMoniter(userid int64, url string) {
	var monitor Monitor
	monitor.Id = userid
	monitor.Url = url
	monitor.Interval = (2 * time.Minute)
	monitor.NextRunTime = time.Now()
	ScheduleMutex.Lock()
	ScheduleMonitor = append(ScheduleMonitor, monitor)
	ScheduleMutex.Unlock()
}

func Scheduler() {
	done := make(chan bool)
	jobs := make(chan *Monitor, 100)
	go LoadWatch(done)
	<-done
	go CheckMoniter(jobs)
	CheckerPool(10, jobs)
}
