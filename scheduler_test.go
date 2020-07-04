package main

import (
	"context"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"path/filepath"

	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"testing"
	"time"
)

type execCommandContextMock struct {
	mock.Mock
}

func (m *execCommandContextMock) CommandContext(ctx context.Context, command string, args ...string) *exec.Cmd {
	i := []interface{}{ctx, command}
	for _, arg := range args {
		i = append(i, arg)
	}
	called := m.Called(i...)
	f := called.Get(0).(func(ctx context.Context, command string, args ...string) *exec.Cmd)
	return f(ctx, command, args...)
}

func fakeExecCommandContext(ctx context.Context, command string, args ...string) *exec.Cmd {
	cs := []string{"-test.run=TestCommandContextHelper", "--", command}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

const DEFAULT_COMMAND_OUTPUT = "command output"

func TestCommandContextHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	os.Stdout.Write([]byte(DEFAULT_COMMAND_OUTPUT))
	os.Exit(0)
}

var DefaultConfigPath = path.Join(filepath.FromSlash("./fixtures"), "test_config.yaml")
var DefaultSchedulerTimeOut, _ = time.ParseDuration("10s")
var DefaultBaseDir = filepath.FromSlash("/tmp")

func init() {
	log.SetOutput(ioutil.Discard)
	execMock := new(execCommandContextMock)
	ExecCommandContext = execMock.CommandContext
	execMock.On("CommandContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		fakeExecCommandContext)
}

func TestRunSchedulerTask(t *testing.T) {
	scheduler, err := NewScheduler(DefaultConfigPath, &DefaultSchedulerTimeOut, DefaultBaseDir, DefaultBaseDir, ".")
	assert.Nil(t, err)
	assert.Len(t, scheduler.Tasks, 5)

	defer os.RemoveAll(scheduler.ResultsDir)

	err = scheduler.RunTask(scheduler.Tasks["test"])
	assert.Nil(t, err)

	files, _ := filepath.Glob(scheduler.BaseDir + "/test*")
	assert.NotEmpty(t, files)
	assert.FileExists(t, files[0])
	output, _ := ioutil.ReadFile(files[0])
	assert.EqualValues(t, output, []byte(DEFAULT_COMMAND_OUTPUT))
}
