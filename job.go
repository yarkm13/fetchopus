package main

import (
	"bufio"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"sync"
)

// JobItem represents a single file to download
type JobItem struct {
	Path   string
	Status int // 0 = pending, 1 = downloaded
}

// Job holds the entire download job
type Job struct {
	SourceURL *url.URL
	TargetDir string
	Items     []JobItem
	mutex     sync.Mutex
	jobFile   string
}

func parseJobFile(filename string) (*Job, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	job := &Job{}
	job.jobFile = filename
	lineNum := 0
	for scanner.Scan() {
		line := scanner.Text()
		if lineNum == 0 {
			job.SourceURL, err = url.Parse(line)
		} else if lineNum == 1 {
			job.TargetDir = line
		} else {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) != 2 {
				continue
			}
			status := 0 // all "in progress" statuses will be reset
			if parts[0] == "1" {
				status = 1
			}
			job.Items = append(job.Items, JobItem{
				Path:   parts[1],
				Status: status,
			})
		}
		lineNum++
	}
	return job, nil
}

func saveJobFile(job *Job) error {
	job.mutex.Lock()
	defer job.mutex.Unlock()

	tmp := job.jobFile + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintln(f, job.SourceURL)
	fmt.Fprintln(f, job.TargetDir)
	for _, item := range job.Items {
		fmt.Fprintf(f, "%d:%s\n", item.Status, item.Path)
	}
	return os.Rename(tmp, job.jobFile)
}

func downloadWorker(job *Job, factory ConnectorFactory, password []byte, wg *sync.WaitGroup, index int) {
	defer wg.Done()

	conn, err := factory.Create(job.SourceURL, password)
	if err != nil {
		log.Printf("[Worker %d] Failed to create connector: %v", index, err)
		return
	}
	defer conn.Close()

	for {
		var item *JobItem

		job.mutex.Lock()
		for i := range job.Items {
			if job.Items[i].Status == 0 {
				job.Items[i].Status = -1 // In progress
				item = &job.Items[i]
				break
			}
		}
		job.mutex.Unlock()

		if item == nil {
			return // No more jobs
		}

		log.Printf("[Worker %d] Downloading: %s", index, item.Path)

		err := conn.DownloadFile(item.Path, job.TargetDir, job.SourceURL.Path)
		if err != nil {
			log.Printf("[Worker %d] Error downloading %s: %v", index, item.Path, err)
			// Retry: set it back to pending
			job.mutex.Lock()
			item.Status = 0
			job.mutex.Unlock()
			continue
		}

		// Mark as done
		job.mutex.Lock()
		item.Status = 1
		job.mutex.Unlock()
	}
}
