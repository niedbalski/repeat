package main

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const DEFAULT_REPORT_PREFIX = "repeat-"
const DEFAULT_ANY_EXIT_CODE = "any"

type Scheduler struct {
	Config          *Config
	GoCronScheduler *gocron.Scheduler
	Pgid            int
	Timeout         *time.Duration
	BaseDir, ResultsDir         string
}

func NewScheduler(configFilename string, timeout *time.Duration, baseDir string, resultsDir string) (*Scheduler, error) {
	var scheduler Scheduler

	log.Infof("Loading collectors from configuration file: %s", configFilename)
	config, err := NewConfigFromFile(configFilename)
	if err != nil {
		return nil, err
	}

	tempDir, err := ioutil.TempDir(baseDir, DEFAULT_REPORT_PREFIX)
	if err != nil {
		return nil, err
	}

	scheduler.BaseDir = tempDir

	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return nil, err
	}

	scheduler.Pgid = pgid
	scheduler.Config = config
	scheduler.GoCronScheduler = gocron.NewScheduler(time.UTC)
	scheduler.ResultsDir = resultsDir

	if timeout != nil {
		log.Infof("Scheduler timeout set to: %f seconds", timeout.Seconds())
		scheduler.Timeout = timeout
	}

	for name, collection := range config.Collections {
		if !collection.RunOnce {
			task, err := NewSchedulerTask(&scheduler, name, pgid, collection)
			if err != nil {
				return nil, err
			}
			log.Infof("Scheduling run of %s collector every %f secs", name, task.RunEvery.Seconds())
			_, err = scheduler.GoCronScheduler.Every(uint64(task.RunEvery.Seconds())).Seconds().StartImmediately().Do(task.Run)
			if err != nil {
				return nil, err
			}
		}
	}
	return &scheduler, nil
}

func (scheduler *Scheduler) Cleanup() (error){
	log.Info("Cleaning up resources")
	reportFileName := filepath.Join(scheduler.ResultsDir,
		fmt.Sprintf("%sreport-%s.tar.gz", DEFAULT_REPORT_PREFIX, time.Now().Format("2006-01-02-15:04")))
	filesToAppend, err := filepath.Glob(filepath.Join(scheduler.ResultsDir, "/*"))
	if err != nil {
		return err
	}

	log.Infof("Creating report tarball at: %s", reportFileName)
	if err = CreateTarball(reportFileName, filesToAppend); err != nil {
		return err
	}

	if err = os.RemoveAll(scheduler.BaseDir); err != nil {
		return err
	}

	return nil
}

func (scheduler *Scheduler) HandleTimeout() {
	timer := time.NewTimer(*scheduler.Timeout)
	defer timer.Stop()

	select {
	case <-timer.C:
		log.Infof("Scheduler timeout (%f) reached, cleaning up and killing process", scheduler.Timeout.Seconds())
		scheduler.Cleanup()
		syscall.Kill(-scheduler.Pgid, syscall.SIGKILL)
	}
}

func (scheduler *Scheduler) Start() error {
	scheduler.GoCronScheduler.StartAsync()

	if scheduler.Timeout != nil {
		go scheduler.HandleTimeout()
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGKILL, syscall.SIGTERM)
	select {
	case <-c:
		scheduler.Cleanup()
		os.Exit(1)
	}

	return nil
}

type SchedulerTask struct {
	Name      string
	RunEvery, Timeout time.Duration
	Config    Collection
	Pgid      int
	Scheduler *Scheduler
}

func NewSchedulerTask(scheduler *Scheduler, name string, pgid int, collection Collection) (*SchedulerTask, error) {
	var task SchedulerTask

	runEvery, err := time.ParseDuration(collection.RunEvery)
	if err != nil {
		return nil, err
	}

	taskTimeout, err := time.ParseDuration(collection.Timeout)
	if err != nil {
		return nil, err
	}

	task.Timeout = taskTimeout
	task.RunEvery = runEvery
	task.Config = collection
	task.Name = name
	task.Pgid = pgid
	task.Scheduler = scheduler
	return &task, nil
}

func IsValidExitCode(allowedExitCodes string, err error) bool {
	if allowedExitCodes == DEFAULT_ANY_EXIT_CODE {
		return true
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		for _, exitCode := range strings.Split(allowedExitCodes, " ") {
			code, _ := strconv.Atoi(exitCode)
			if code == exitError.ExitCode() {
				return true
			}
		}
	}
	return false
}

func (task *SchedulerTask) Run() {
	var timeout time.Duration

	if task.Timeout <= 0 {
		timeout = *task.Scheduler.Timeout
	} else {
		timeout = task.Timeout
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "bash", "-c", task.Config.Command)
	cmd.Dir = task.Scheduler.BaseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: task.Pgid}

	log.Infof("Running command for collector %s", task.Name)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		log.Warnf("Collector: %s, timed out after %f secs (cancelled)", task.Name, timeout.Seconds())
		return
	}

	if err != nil && !IsValidExitCode(task.Config.ExitCodes, err) {
		log.Errorf("Command for collector %s exited with exit code: %s - (not allowed by exit-codes config)",
			task.Name, err.Error())
		return
	}

	outputFileName := filepath.Join(task.Scheduler.BaseDir, fmt.Sprintf("%s-%s", task.Name, time.Now().Format("2006-01-02-15:04:05")))

	if err = ioutil.WriteFile(outputFileName, output, 0750); err != nil {
		log.Errorf("Error storing collection results for %s, on file: %s", task.Name, outputFileName)
		return
	}

	log.Infof("Command for collector %s, successfully ran, stored results into file: %s", task.Name, outputFileName)
	return
}
