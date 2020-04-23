package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"testing"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"time"
)

type execCommandContextMock struct {
	mock.Mock
}

type execCommandMock struct {
	mock.Mock
}

type writeFileMock struct {
	mock.Mock
}

type tempDirMock struct {
	mock.Mock
}

func (_m *writeFileMock) WriteFile(filename string, data []byte, perm os.FileMode) error {
	ret := _m.Called(filename, data, perm)

	var r0 error
	if rf, ok := ret.Get(0).(func(string, []byte, os.FileMode) error); ok {
		r0 = rf(filename, data, perm)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

func (_m *tempDirMock) TempDir(_a0 string, _a1 string) (string, error) {
	ret := _m.Called(_a0, _a1)

	var r0 string
	if rf, ok := ret.Get(0).(func(string, string) string); ok {
		r0 = rf(_a0, _a1)
	} else {
		r0 = ret.Get(0).(string)
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(string, string) error); ok {
		r1 = rf(_a0, _a1)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

func (m *execCommandMock) Command(command string, args ...string) *exec.Cmd {
	i := []interface{}{command}
	for _, arg := range args {
		i = append(i, arg)
	}
	return m.Called(i...).Get(0).(func(string, ...string) *exec.Cmd)(command, args...)
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
	cs := []string{"-test.run=TestLoadDriverPluginsCallsConfigHelper", "--", command}
	cs = append(cs, args...)
	cmd := exec.CommandContext(ctx, os.Args[0], cs...)
	cmd.Env = []string{"GO_WANT_HELPER_PROCESS=1"}
	return cmd
}

var output = "command output"
func TestLoadDriverPluginsCallsConfigHelper(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	json.NewEncoder(os.Stdout).Encode(output)
	os.Exit(0)
}

var configPath = path.Join("./fixtures", "test_config.yaml")
var schedulerTimeout, _ = time.ParseDuration("10s")


func TestNewScheduler(t *testing.T) {
	TempDirMock := new(tempDirMock)
	Tempdir = TempDirMock.TempDir
	TempDirMock.On("TempDir", mock.Anything, mock.Anything).Return(".", nil)

	writeFileMock := new(writeFileMock)
	WriteFile = writeFileMock.WriteFile
	writeFileMock.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)

	scheduler, err := NewScheduler(configPath, &schedulerTimeout, ".", ".")
	TempDirMock.AssertNumberOfCalls(t, "TempDir", 1)

	assert.NotNil(t, scheduler)
	assert.Nil(t, err)
	assert.Equal(t, scheduler.Timeout.Seconds(), schedulerTimeout.Seconds())
	assert.Contains(t, scheduler.Tasks, "test", "test1", "test2", "test3", "test4")
	scheduler, err = NewScheduler(configPath + "wrong", &schedulerTimeout, ".", ".")
	assert.NotNil(t, err)
}
func TestScheduler_RunTask(t *testing.T) {
	execMock := new(execCommandContextMock)
	//execCommandMock := new(execCommandMock)

	//ExecCommand = execCommandMock.Command
	ExecCommandContext = execMock.CommandContext

	execMock.On("CommandContext", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(
		fakeExecCommandContext)

	writeFileMock := new(writeFileMock)
	WriteFile = writeFileMock.WriteFile

	writeFileMock.On("WriteFile", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	//Run(func(args mock.Arguments) {
	//	fmt.Println(args)
	//}).Return(nil)

	//r.
	//	On("write", mock.Anything).
	//	Run(func(args mock.Arguments) {
	//		fmt.Println("called")
	//
	//	}).
	//	Return(1, nil)

	scheduler, _ := NewScheduler(configPath, &schedulerTimeout, ".", ".")
	outputFilename, err := scheduler.RunTask(scheduler.Tasks["test"])

	writeFileMock.AssertCalled(t, "WriteFile", mock.Anything, mock.Anything, mock.Anything)

	fmt.Println(outputFilename, err)

	//assert.Nil(t, err)
	//assert.FileExists(t, outputFilename)
}

