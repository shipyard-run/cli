package container

import (
	"os"
	"testing"

	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/testutils"
	"github.com/stretchr/testify/require"
)

func init() {
	config.RegisterResource(TypeSidecar, &Sidecar{}, &Provider{})
}

func TestSidecarProcessSetsAbsolute(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	c := &Sidecar{
		ResourceMetadata: types.ResourceMetadata{ResourceFile: "./"},
		Volumes: []Volume{
			{
				Source:      "./",
				Destination: "./",
			},
		},
	}

	c.Process()

	require.Equal(t, wd, c.Volumes[0].Source)
}

func TestSidecarLoadsValuesFromState(t *testing.T) {
	testutils.SetupState(t, `
{
  "blueprint": null,
  "resources": [
	{
			"resource_id": "resource.sidecar.test",
      "resource_name": "test",
      "resource_type": "sidecar",
			"container_name": "fqdn.mine"
	}
	]
}`)

	docs := &Sidecar{
		ResourceMetadata: types.ResourceMetadata{
			ResourceFile: "./",
			ResourceID:   "resource.sidecar.test",
		},
	}

	err := docs.Process()
	require.NoError(t, err)

	require.Equal(t, "fqdn.mine", docs.ContainerName)
}
