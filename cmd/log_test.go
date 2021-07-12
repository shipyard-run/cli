package cmd

import (
	"os"
	"syscall"
	"testing"
	"time"
	
	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	
	"github.com/shipyard-run/shipyard/pkg/shipyard"
	"github.com/shipyard-run/shipyard/pkg/utils"
)

// These tests only run successfully when a blueprint is running,
// It tests whether the streamed stack/cluster logs are greater than the specified size.
// not sure how to add this to `test_feature`

// single_k3s_cluster
const bluePrintDockerLogSize int64 = 10000 // Kb
const bluePrintClusterLogSize int64 = 10000 // Kb
const bluePrintListSize = 10 // Kb

// signal cli exit
const UserInterruptTime = 3 * time.Second

// setupFile sets up a tmp *os.File to redirect cli logs
func mockStdOut(t *testing.T) *os.File {
	cwd, _ := os.Getwd()
	tmpFile, err := os.CreateTemp(cwd, ".tmp.logs.")
	assert.NoError(t, err)
	return tmpFile
}
// checks shipyard config exists and env is valid
func checkK8sRunning(t *testing.T) bool {
	if assert.FileExists(t, utils.StatePath(),"No docker+k8 blueprint is running," +
		"`shipyard run github.com/shipyard-run/shipyard/examples/single_k3s_cluster`"){
		if assert.Contains(t, os.Getenv("KUBECONFIG"), ".yaml" ,
			"KUBECONFIG not set"){
			return true
		}
	}
	return false
}

// runCmdThenInterruptIt Tests whether output from cli utility is greater than the
// expectedSize
func runCmdThenInterruptIt(t *testing.T, logs *cobra.Command, tmpFile *os.File, expectedSize int64){
	defer func(tmpFile *os.File) {
		assert.NoError(t, os.Remove(tmpFile.Name()))
	}(tmpFile)
	// user interrupt, to stop tailing logs
	go func() {
		<- time.After(UserInterruptTime)
		err := syscall.Kill(syscall.Getpid(), syscall.SIGINT)
		assert.NoError(t, err)
	}()
	// execute the cli log command, which runs until interrupt
	err := logs.Execute()
	assert.NoError(t, err)

	// not sure how else to verify whether logs worked on not
	stats, _ := os.Stat(tmpFile.Name())
	assert.NotNil(t, stats)
	
	// fmt.Println(stats.Size())
	assert.Greater(t, stats.Size(), expectedSize, "Response size is lower than expected")
}

func testCommand(t *testing.T, engine shipyard.Engine, args []string, expectedSize int64) {
	t.Parallel()
	
	listFile := mockStdOut(t)
	
	// `shipyard log`
	listCmd := logCmd(listFile, engine)
	
	// add cli args
	listCmd.SetArgs(args)
	
	// run cli and verify output size
	runCmdThenInterruptIt(t, listCmd, listFile, expectedSize)
}

// make test_unit will fail here if k8s blueprint isn't running
// `shipyard run github.com/shipyard-run/shipyard/examples/single_k3s_cluster`
// `export $kubecofig..`
// `go test log_test.go log.go util.go -v -cover -race `
// {{uncomment}} func TestLogCmd(t *testing.T) {
func testLogCmd(t *testing.T) {
	// t.Parallel()
	
	engine, err := shipyard.New(hclog.NewNullLogger())
	assert.NoError(t, err)
	
	t.Run("Test `shipyard log`", func(t *testing.T) {
		testCommand(t, engine, nil, bluePrintListSize)
	})
	
	t.Run("Test `shipyard log badcommand`", func(t *testing.T) {
		testCommand(t, engine, []string{"something"}, 4)
	})
	
	t.Run("Test `shipyard log containers`", func(t *testing.T) {
		testCommand(t, engine, []string{"containers"}, bluePrintClusterLogSize)
	})
	
	if checkK8sRunning(t){
		t.Run("Test `shipyard log kubernetes`", func(t *testing.T) {
			testCommand(t, engine, []string{"kubernetes"}, bluePrintDockerLogSize)
		})
	}

}
