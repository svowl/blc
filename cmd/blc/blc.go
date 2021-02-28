package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"

	"github.com/gorilla/mux"
	"github.com/nightlyone/lockfile"
	"github.com/robfig/cron/v3"
	"gopkg.in/gcfg.v1"

	"blc/pkg/api"
	"blc/pkg/auth"
	"blc/pkg/conf"
	"blc/pkg/crawler"
	"blc/pkg/logger"
	"blc/pkg/report"
	"blc/pkg/wsserver"
)

const lockFile = ".lock"
const configFile = "./config.ini"

var crawlers map[int]*crawler.Service

func main() {
	// Set lock
	lock := lock()
	defer func() {
		if err := lock.Unlock(); err != nil {
			log.Fatalf("Cannot unlock %q, reason: %v", lock, err)
		}
	}()

	log.Println("Lock:", lock)

	// Read the config file
	var cfg conf.Config
	err := gcfg.ReadFileInto(&cfg, configFile)
	if err != nil {
		log.Fatalf("Failed to parse gcfg data: %s", err)
	}

	logger := logger.New(os.Stdout, os.Stderr)

	// Create auth object and initialize it with users list file and token TTL (in minutes)
	a := auth.New(cfg.Auth.Userslist, 60)
	router := mux.NewRouter()

	crawlers = make(map[int]*crawler.Service, 1)

	chReport := make(chan *crawler.Service)

	var mx sync.Mutex
	go processFinish(chReport, &cfg, &mx)

	s := wsserver.New(logger, crawlers, router, a, cfg.Crawler.Delay, chReport)
	s.Endpoints()

	api := api.New(logger, crawlers, router, a)
	api.Endpoints()

	if len(cfg.Schedule) > 0 {
		c := cron.New()
		i := 1
		for schedName, sched := range cfg.Schedule {
			log.Printf("Schedule crawler process %q", schedName)
			ID := i
			c.AddFunc(sched.Cron, func() {
				crawlers[ID] = crawler.New(ID, cfg.Crawler.Delay, chReport, logger)
				go s.PublishMessages(crawlers[ID].ChResults)
				crawlers[ID].Scan(sched.URL, sched.Depth, sched.SessionName, sched.ExcludedURL)
			})
			i++
			if i >= 100 {
				// Allow only 100 scheduled jobs
				break
			}
		}
		c.Start()
	}

	router.PathPrefix("/").Handler(http.StripPrefix("/", http.FileServer(http.Dir("./webapp"))))

	// Run HTTP server
	if err := http.ListenAndServe(cfg.Server.Addr, router); err != nil {
		log.Fatal(err)
	}
}

// processFinish performs final operations after crawling is finished: save report in file and send to email
func processFinish(chReport chan *crawler.Service, cfg *conf.Config, mx *sync.Mutex) {
	for {
		s := <-chReport
		data := report.JSONData{
			TimeElapsed:  s.TimeElapsed,
			TimeFinished: s.TimeFinished,
			TotalLinks:   len(s.Processed),
			URLs:         s.URLs,
			Errors:       s.Errors,
		}
		mx.Lock()
		fileName, err := report.Save(&data)
		if err != nil {
			log.Printf("Report was not saved: %v", err)
		} else {
			log.Printf("Report saved in %s", fileName)
		}
		csvFileName, err := report.SaveCSV(&data)
		if err != nil {
			log.Printf("CSV report was not saved: %v", err)
		} else {
			log.Printf("CSV report saved in %s", csvFileName)
		}
		if err := report.Send(cfg.SMTP, data); err != nil {
			log.Printf("Report failed to send: %v", err)
		} else {
			log.Printf("Report has been successfully sent to %s", cfg.To)
		}
		if err := report.CleanReports(cfg.Reports.MaxReportsToStore); err != nil {
			log.Printf("Error of clean up reports directory: %v", err)
		}
		// Repeat final message to be sure the reports block on HTML page is refreshed with correct data
		s.ChResults <- crawler.ScanResult{ProgressState: crawler.STOPPED, ID: s.ID, TotalLinks: len(s.Processed), TotalErrors: len(s.Errors), URLs: s.URLs}
		delete(crawlers, s.ID)
		mx.Unlock()
	}
}

// Sets the lockfile
func lock() lockfile.Lockfile {
	var curDir, err = os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	lock, err := lockfile.New(filepath.Join(curDir, lockFile))
	if err != nil {
		log.Fatalf("Cannot init lock. reason: %v", err)
	}
	if err = lock.TryLock(); err != nil {
		log.Fatalf("Cannot lock %q, reason: %v", lock, err)
	}
	return lock
}
