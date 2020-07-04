package main

import (
	"context"
	"fmt"
	"github.com/go-co-op/gocron"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const DEFAULT_REPORT_PREFIX = "repeat-"
const DEFAULT_ANY_EXIT_CODE = "any"

var ExecCommand = exec.Command
var ExecCommandContext = exec.CommandContext

type Scheduler struct {
	Config                     *Config
	DBStorage                  *DBStorage
	GoCronScheduler            *gocron.Scheduler
	Pgid                       int
	Timeout                    *time.Duration
	DBDir, BaseDir, ResultsDir string
	Tasks                      map[string]*SchedulerTask
}

type SchedulerTask struct {
	Name              string
	RunEvery, Timeout time.Duration
	Config            Collection
	Pgid              int
	Command           string
	Job               *gocron.Job
	BaseDir           string
	DBStorage         *DBStorage
}

var Tempdir = ioutil.TempDir

func NewScheduler(configFilename string, timeout *time.Duration, baseDir, resultsDir, dbDir string) (*Scheduler, error) {
	var scheduler Scheduler
	var t time.Location

	log.Infof("Loading collectors from configuration file: %s", configFilename)
	config, err := NewConfigFromFile(configFilename)
	if err != nil {
		return nil, err
	}

	tempDir, err := Tempdir(baseDir, DEFAULT_REPORT_PREFIX)
	if err != nil {
		return nil, err
	}

	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return nil, err
	}

	scheduler.BaseDir = tempDir
	scheduler.Pgid = pgid
	scheduler.Config = config
	scheduler.GoCronScheduler = gocron.NewScheduler(&t)
	scheduler.ResultsDir = resultsDir
	scheduler.Tasks = make(map[string]*SchedulerTask)
	scheduler.DBDir = dbDir

	storage, err := NewDBStorage(scheduler.BaseDir)
	if err != nil {
		return nil, err
	}
	scheduler.DBStorage = storage

	if timeout != nil {
		log.Infof("Scheduler timeout set to: %f seconds", timeout.Seconds())
		scheduler.Timeout = timeout
	}

	if err = scheduler.LoadTasks(); err != nil {
		return nil, err
	}

	return &scheduler, nil
}

func (scheduler *Scheduler) LoadTasks() error {
	var task *SchedulerTask
	var err error

	for name, collection := range scheduler.Config.Collections {
		if task, err = NewSchedulerTask(name, scheduler.BaseDir, scheduler.Pgid, scheduler.DBDir, collection, scheduler.DBStorage); err != nil {
			return err
		}
		scheduler.Tasks[name] = task
	}
	return nil
}

func (scheduler *Scheduler) TarballReport() error {
	reportFileName := filepath.Join(scheduler.ResultsDir,
		fmt.Sprintf("%sreport-%s.tar.gz", DEFAULT_REPORT_PREFIX, time.Now().Format("2006-01-02-15-04")))
	filesToAppend, err := filepath.Glob(filepath.Join(scheduler.BaseDir, "/*"))
	if err != nil {
		return err
	}

	log.Infof("Creating report tarball at: %s", reportFileName)
	if err = CreateTarball(reportFileName, filesToAppend); err != nil {
		return err
	}
	return nil
}

func (scheduler *Scheduler) Cleanup() error {
	log.Info("Cleaning up resources")
	scheduler.GoCronScheduler.Clear()
	if err := scheduler.TarballReport(); err != nil {
		return err
	}

	log.Debugf("Removing base directory: %s", scheduler.BaseDir)
	if err := os.RemoveAll(scheduler.BaseDir); err != nil {
		return err
	}
	return nil
}

func (scheduler *Scheduler) HandleTimeout() {
	timer := time.NewTimer(*scheduler.Timeout)
	defer timer.Stop()

	for range timer.C {
		log.Infof("Scheduler timeout (%f) reached, cleaning up and killing process", scheduler.Timeout.Seconds())
		if err := scheduler.Cleanup(); err != nil {
			log.Error(err)
		}
		if err := syscall.Kill(-scheduler.Pgid, syscall.SIGKILL); err != nil {
			log.Errorf("Error killing process group: %d", scheduler.Pgid)
			os.Exit(-1)
		}
	}
}

func (scheduler *Scheduler) RemoveTask(name string) {
	if task, ok := scheduler.Tasks[name]; !ok {
		log.Debugf("Not found available task with name: %s in scheduler", name)
	} else {
		scheduler.GoCronScheduler.RemoveByReference(task.Job)
		log.Debugf("Removed task name: %s from scheduler", name)
	}
}

var WriteFile = ioutil.WriteFile

func (scheduler *Scheduler) RunTask(task *SchedulerTask) error {
	var output []byte
	var err error

	// If its a run-once job, only run it once and then discard the job from the scheduler.
	if task.Config.RunOnce {
		scheduler.RemoveTask(task.Name)
	}

	if task.Timeout > 0 {
		output, err = RunWithTimeout(task)
	} else {
		output, err = RunWithoutTimeout(task)
	}

	if err != nil && !task.IsValidExitCode(err) {
		errMsg := fmt.Errorf("Command for collector %s exited with exit code: %s - (not allowed by exit-codes config)",
			task.Name, err.Error())
		log.Error(errMsg)
		return errMsg
	}

	switch task.Config.Store {
	case "database":
		{
			return task.StoreResultsToDB(output)
		}
	case "file":
		{
			return task.StoreResultsToFile(output)
		}
	}
	return nil
}

func (scheduler *Scheduler) Start() error {

	for name, task := range scheduler.Tasks {
		log.Infof("Scheduling run of %s collector every %f secs", name, task.RunEvery.Seconds())
		job, err := scheduler.GoCronScheduler.Every(uint64(task.RunEvery.Seconds())).Seconds().StartImmediately().Do(scheduler.RunTask, task)
		if err != nil {
			return err
		}
		scheduler.Tasks[name].Job = job
	}

	scheduler.GoCronScheduler.StartAsync()

	if *scheduler.Timeout > 0 {
		go scheduler.HandleTimeout()
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	for range c {
		if err := scheduler.Cleanup(); err != nil {
			log.Errorf("Error during the cleanup phase: %s", err)
		}
		os.Exit(1)
	}
	return nil
}

var TempFile = ioutil.TempFile

func NewSchedulerTask(name, baseDir string, pgid int, dbDir string, collection Collection, dBStorage *DBStorage) (*SchedulerTask, error) {
	var task SchedulerTask
	var command string

	runEvery, err := time.ParseDuration(collection.RunEvery)
	if err != nil {
		return nil, err
	}

	taskTimeout, err := time.ParseDuration(collection.Timeout)
	if err != nil {
		return nil, err
	}

	if collection.RunOnce && runEvery > 0 {
		return nil, fmt.Errorf("task: %s must be defined as run-once or run-every, not both", name)
	}

	if collection.Script != "" {
		fd, err := TempFile(baseDir, "run-script-")
		if err != nil {
			return nil, err
		}
		if err = fd.Chmod(0700); err != nil {
			return nil, err
		}
		if _, err = fd.WriteString(collection.Script); err != nil {
			return nil, err
		}

		fd.Close()
		command = fd.Name()
	} else {
		command = collection.Command
	}

	task.BaseDir = baseDir
	task.Command = command
	task.Timeout = taskTimeout
	task.RunEvery = runEvery
	task.Config = collection
	task.Name = name
	task.Pgid = pgid
	task.DBStorage = dBStorage
	return &task, nil
}

func (task *SchedulerTask) IsValidExitCode(err error) bool {
	if task.Config.ExitCodes == DEFAULT_ANY_EXIT_CODE {
		return true
	}
	if exitError, ok := err.(*exec.ExitError); ok {
		for _, exitCode := range strings.Split(task.Config.ExitCodes, " ") {
			code, _ := strconv.Atoi(exitCode)
			if code == exitError.ExitCode() {
				return true
			}
		}
	}
	return false
}

func (task *SchedulerTask) StoreResultsToDB(results []byte) error {
	tableName := strings.ToLower(task.Name)
	for _, line := range strings.Split(string(results), "\n") {
		values := strings.Split(line, task.Config.Database.MapValues.Separator)
		fields := task.Config.Database.MapValues.Fields
		task.DBStorage.CreateTable(tableName, fields)
		if err := task.DBStorage.CreateRecord(tableName, fields, values); err != nil {
			return err
		}
	}
	log.Infof("Command for collector %s, successfully ran, stored results into database, table: %s", task.Name, tableName)
	return nil
}

func (task *SchedulerTask) StoreResultsToFile(results []byte) error {
	outputFileName := filepath.Join(task.BaseDir, fmt.Sprintf("%s-%s", task.Name, time.Now().Format("2006-01-02-15:04:05")))
	if err := WriteFile(outputFileName, results, 0750); err != nil {
		log.Errorf("Error storing collection results for %s, on file: %s", task.Name, outputFileName)
		return err
	}
	log.Infof("Command for collector %s, successfully ran, stored results into file: %s", task.Name, outputFileName)
	return nil
}

func RunWithTimeout(task *SchedulerTask) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), task.Timeout)
	defer cancel()
	cmd := ExecCommandContext(ctx, "bash", "-c", task.Command)
	cmd.Dir = task.BaseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: task.Pgid}

	log.Infof("Running command for collector %s", task.Name)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		log.Warnf("Collector: %s, timed out after %f secs (cancelled)", task.Name, task.Timeout.Seconds())
		return nil, nil
	}
	return output, err
}

func RunWithoutTimeout(task *SchedulerTask) ([]byte, error) {
	cmd := ExecCommand("bash", "-c", task.Command)
	cmd.Dir = task.BaseDir
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true, Pgid: task.Pgid}
	log.Infof("Running command for collector %s", task.Name)
	return cmd.CombinedOutput()
}
