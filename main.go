package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"sync"
	"time"
	//"golang.org/x/crypto/ssh"
	//"github.com/pkg/sftp"
)

func main() {
	urlFlag := flag.String("url", "", "Source URL (ftp:// or scp://)")
	targetFlag := flag.String("target-dir", "", "Target directory")
	threadsFlag := flag.Int("threads", 4, "Number of download threads")
	jobFlag := flag.String("job", "", "Resume from job file")
	flag.Parse()

	var job *Job
	var jobFile string
	var err error
	var u *url.URL
	var connectorFactory ConnectorFactory

	var password []byte
	if *jobFlag != "" {
		jobFile = *jobFlag
		job, err = parseJobFile(jobFile)
		if err != nil {
			log.Fatalf("Error reading job file: %v", err)
		}
		password = askPassword()
		u = job.SourceURL // Use source URL from job file instead of expecting it from command line
		if u == nil {
			log.Fatalf("Invalid URL in job file")
		}
	} else {
		if *urlFlag == "" || *targetFlag == "" {
			log.Fatal("Missing required parameters: --url, --target-dir")
		}
		u, err = url.Parse(*urlFlag)
		if err != nil {
			log.Fatalf("Invalid URLL: %v", err)
		}
		passwordStr, passSet := u.User.Password()
		if !passSet {
			password = askPassword()
			fmt.Println("Password saved")
		} else {
			password = make([]byte, len(passwordStr))
			copy(password, []byte(passwordStr))
			passwordStr = ""
		}
		job = &Job{
			SourceURL: u,
			TargetDir: *targetFlag,
			jobFile:   time.Now().Format("20060102150405") + ".dljob",
		}
	}

	var connector Connector
	connectorFactory = getConnectorFactory(u)
	if connectorFactory == nil {
		log.Fatalf("No connector available for scheme: %s", u.Scheme)
	}
	connector, err = connectorFactory.Create(u, password)
	if err != nil {
		log.Fatalf("Connection error: %v", err)
	}
	defer connector.Close()
	if len(job.Items) < 1 {
		files, err := connector.ListFilesRecursively(u.Path)
		if err != nil {
			log.Fatalf("Error listing files: %v", err)
		}
		for _, f := range files {
			job.Items = append(job.Items, JobItem{Path: f, Status: 0})
		}
	}

	saveJobFile(job)

	if !promptToContinue(job) {
		log.Printf("Jobfile saved as %s. You can edit path there and resume the download providing --job %s option.", job.jobFile, job.jobFile)
		return
	}

	// Securely clear password when it's no longer needed
	defer func() {
		secureWipe(password)
		password = nil
	}()

	// Background job saver
	ctx, cancelAutosave := context.WithCancel(context.Background())
	defer cancelAutosave()

	go func() {
		ticker := time.NewTicker(time.Second * 2)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				saveJobFile(job)
			}
		}
	}()

	var wg sync.WaitGroup
	for i := 0; i < *threadsFlag; i++ {
		wg.Add(1)
		go downloadWorker(job, connectorFactory, password, &wg, i+1)
	}
	wg.Wait()
	cancelAutosave()
	saveJobFile(job)
	log.Println("All downloads completed.")
}
