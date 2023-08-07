package container

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/jumppad-labs/jumppad/pkg/clients"
	"github.com/jumppad-labs/jumppad/pkg/clients/container/mocks"
	cmocks "github.com/jumppad-labs/jumppad/pkg/clients/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestContainerLogsCalled(t *testing.T) {
	md := &mocks.Docker{}
	md.On("ServerVersion", mock.Anything).Return(types.Version{}, nil)
	md.On("ContainerLogs", mock.Anything, mock.Anything, mock.Anything).Return(
		ioutil.NopCloser(bytes.NewBufferString("test")),
		fmt.Errorf("boom"),
	)

	md.On("Info", mock.Anything).Return(types.Info{Driver: StorageDriverOverlay2}, nil)

	mic := &cmocks.ImageLog{}

	dt := NewDockerTasks(md, mic, &clients.TarGz{}, clients.NewTestLogger(t))

	rc, err := dt.ContainerLogs("123", true, true)
	assert.NotNil(t, rc)
	assert.Error(t, err)
}
