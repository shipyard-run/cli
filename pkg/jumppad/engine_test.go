package jumppad

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jumppad-labs/hclconfig"
	"github.com/jumppad-labs/hclconfig/types"
	"github.com/jumppad-labs/jumppad/pkg/clients/logger"
	"github.com/jumppad-labs/jumppad/pkg/config"
	"github.com/jumppad-labs/jumppad/pkg/config/mocks"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/cache"
	"github.com/jumppad-labs/jumppad/pkg/config/resources/container"
	"github.com/jumppad-labs/jumppad/pkg/jumppad/constants"
	"github.com/jumppad-labs/jumppad/pkg/utils"
	"github.com/jumppad-labs/jumppad/testutils"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var lock = sync.Mutex{}

func setupTests(t *testing.T, returnVals map[string]error) (*EngineImpl, *mocks.Providers) {
	return setupTestsBase(t, returnVals, "")
}

func setupTestsWithState(t *testing.T, returnVals map[string]error, state string) (*EngineImpl, *mocks.Providers) {
	return setupTestsBase(t, returnVals, state)
}

func setupTestsBase(t *testing.T, returnVals map[string]error, state string) (*EngineImpl, *mocks.Providers) {
	l := logger.NewTestLogger(t)

	log.SetOutput(l.StandardWriter())

	pm := mocks.NewProviders(returnVals)
	pm.On("GetProvider", mock.Anything)

	e := &EngineImpl{
		log:       l,
		providers: pm,
	}

	testutils.SetupState(t, state)

	return e, pm
}

func testLoadState(t *testing.T, e *EngineImpl) *hclconfig.Config {
	c, err := config.LoadState()
	require.NoError(t, err)

	return c
}

func getResourceFromMock(mp *mocks.Providers, index int) types.Resource {
	p := mp.Providers[index]
	r := testutils.GetCalls(&p.Mock, "Init")[0].Arguments[0]

	return r.(types.Resource)
}

func getMetaFromMock(mp *mocks.Providers, index int) types.Meta {
	return *getResourceFromMock(mp, index).Metadata()
}

func TestApplyWithSingleFile(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, e.config.Resources, 7) // 6 resources in the file plus the image cache

	// Check the provider was called for each resource
	require.ElementsMatch(t,
		[]string{
			"default",
			"onprem",
			"default",
			"consul_config",
			"port_range",
			"version",
		},
		[]string{
			getMetaFromMock(mp, 0).Name,
			getMetaFromMock(mp, 1).Name,
			getMetaFromMock(mp, 2).Name,
			getMetaFromMock(mp, 3).Name,
			getMetaFromMock(mp, 4).Name,
			getMetaFromMock(mp, 5).Name,
		},
	)
}

func TestApplyWithSingleFileWithEvents(t *testing.T) {
	e, mp := setupTests(t, nil)

	events, err := e.Events()
	require.NoError(t, err)

	parsedEventCount := 0
	creatingEventCount := 0
	createdEventCount := 0

	go func() {
		for event := range events {
			require.Contains(t, []constants.LifecycleEvent{constants.LifecycleEventParsed, constants.LifecycleEventCreating, constants.LifecycleEventCreated}, event.Status)
			switch event.Status {
			case constants.LifecycleEventParsed:
				parsedEventCount += 1
			case constants.LifecycleEventCreating:
				creatingEventCount += 1
			case constants.LifecycleEventCreated:
				createdEventCount += 1
			}
		}
	}()

	_, err = e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, e.config.Resources, 7) // 6 resources in the file plus the image cache

	// Check the provider was called for each resource
	require.ElementsMatch(t,
		[]string{
			"default",
			"onprem",
			"default",
			"consul_config",
			"port_range",
			"version",
		},
		[]string{
			getMetaFromMock(mp, 0).Name,
			getMetaFromMock(mp, 1).Name,
			getMetaFromMock(mp, 2).Name,
			getMetaFromMock(mp, 3).Name,
			getMetaFromMock(mp, 4).Name,
			getMetaFromMock(mp, 5).Name,
		},
	)

	require.Equal(t, 6, parsedEventCount)
	require.Equal(t, 6, createdEventCount)
	require.Equal(t, 6, creatingEventCount)
	require.Empty(t, events)
}

func TestApplyAddsImageCache(t *testing.T) {
	e, _ := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	dc := e.ResourceCountForType(cache.TypeImageCache)
	require.Equal(t, 1, dc)
}

func TestApplyAddsNetworksToImageCache(t *testing.T) {
	e, _ := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	dc := e.ResourceCountForType(cache.TypeImageCache)
	require.Equal(t, 1, dc)

	r, err := e.config.FindResource("resource.image_cache.default")
	require.NoError(t, err)

	// network should be added as a dependency
	require.Equal(t, "resource.network.onprem", r.GetDependencies()[0])
}

func TestApplyAddsCustomRegistriesToImageCache(t *testing.T) {
	e, _ := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/registries")
	require.NoError(t, err)

	dc := e.ResourceCountForType(cache.TypeImageCache)
	require.Equal(t, 1, dc)

	dc = e.ResourceCountForType(cache.TypeRegistry)
	require.Equal(t, 2, dc)

	r, err := e.config.FindResource("resource.image_cache.default")
	require.NoError(t, err)
	require.Len(t, r.(*cache.ImageCache).Registries, 2)
}

func TestApplyWithSingleFileAndVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.ApplyWithVariables(context.Background(), "../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	require.NoError(t, err)
	require.Len(t, e.config.Resources, 7) // 6 resources in the file plus the image cache

	// then the container should be created
	require.Equal(t, "consul", getMetaFromMock(mp, 6).Name)

	// finally the provider for the image cache should be updated
	require.Equal(t, "consul_addr", getMetaFromMock(mp, 7).Name)

	// check the variable has overridden the image
	cont := getResourceFromMock(mp, 6).(*container.Container)
	require.Equal(t, "consul:1.8.1", cont.Image.Name)
}

func TestApplyCallsProviderCreateForEachProvider(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/single_k3s_cluster")
	require.NoError(t, err)

	// should have call create for each resource in the config
	// and once for ImageCache that is manually added.
	// for every network that is created refresh is called on the image cache
	rc := len(e.config.Resources)
	testAssertMethodCalled(t, mp, "Create", rc)
	testAssertMethodCalled(t, mp, "Refresh", 1)

	// the state should also contain 11 resources
	sf := testLoadState(t, e)
	require.Equal(t, 11, sf.ResourceCount())
}

func TestApplyDoesNotCallsProviderCreateWhenInState(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	_, err := e.Apply(context.Background(), "../../examples/single_file")
	require.NoError(t, err)

	// should only be called for resources that are not in the state
	testAssertMethodCalled(t, mp, "Create", 5)

	// the state should also contain 7 resources
	sf := testLoadState(t, e)
	require.Equal(t, 7, sf.ResourceCount())
}

func TestApplyRemovesItemsInStateWhenNotInFiles(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	_, err := e.Apply(context.Background(), "../../examples/single_file")
	require.NoError(t, err)

	// should only be called for resources that are not in the state
	testAssertMethodCalled(t, mp, "Create", 5)

	// should remove items not in files
	testAssertMethodCalled(t, mp, "Destroy", 2)

	// the state should also contain 7 resources
	sf := testLoadState(t, e)
	require.Equal(t, 7, sf.ResourceCount())
}

func TestApplyNotCallsProviderCreateForDisabledResources(t *testing.T) {
	e, mp := setupTests(t, nil)

	// contains 3 resources one is disabled
	_, err := e.Apply(context.Background(), "../../examples/disabled")
	require.NoError(t, err)

	// should have call create for non disabled resources
	testAssertMethodCalled(t, mp, "Create", 3) // ImageCache is always created

	// disabled resources should still be added to the state
	sf := testLoadState(t, e)

	// should contain 3 from the config plus the image cache
	require.Equal(t, 4, sf.ResourceCount())

	// the resource should be in the state but there should be no status
	r, err := sf.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.Nil(t, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplyShouldNotAddDuplicateDisabledResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	// contains 2 resources one is disabled
	_, err := e.Apply(context.Background(), "../../examples/disabled")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Create", 1) // ImageCache is always created

	// disabled resources should still be added to the state
	sf := testLoadState(t, e)

	// should contain 3 from the config plus the image cache
	// should not duplicate the disabled container as this already exists in the
	// state
	require.Equal(t, 4, sf.ResourceCount())

	// the status should be set to disabled
	r, err := sf.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.Equal(t, constants.StatusDisabled, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplySetsCreatedStatusForEachResource(t *testing.T) {
	e, mp := setupTests(t, nil)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Equal(t, 7, e.config.ResourceCount())

	// should only call create and destroy for the cache as this is pending update
	testAssertMethodCalled(t, mp, "Create", 7) // ImageCache is always created

	sf := testLoadState(t, e)

	r, err := sf.FindResource("resource.container.consul")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	r, err = sf.FindResource("resource.network.onprem")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	r, err = sf.FindResource("resource.template.consul_config")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])
}

func TestApplyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTests(t, map[string]error{"onprem": fmt.Errorf("boom")})

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.Error(t, err)

	// should have call create for each provider
	// there are two top level config items, template and network
	// network will fail
	// ImageCache should always be created
	testAssertMethodCalled(t, mp, "Create", 5)

	sf := testLoadState(t, e)

	// should set failed status for network
	r, err := sf.FindResource("resource.network.onprem")
	require.NoError(t, err)
	require.Equal(t, constants.StatusFailed, r.Metadata().Properties[constants.PropertyStatus])

	// should set created status for template
	r, err = sf.FindResource("resource.template.consul_config")
	require.NoError(t, err)
	require.Equal(t, constants.StatusCreated, r.Metadata().Properties[constants.PropertyStatus])

	// container should not be in the state
	_, err = sf.FindResource("resource.container.consul")
	require.Error(t, err)
}

func TestApplyCallsProviderDestroyAndCreateForFailedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, failedState)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 7) // ImageCache is always created
}

func TestApplyCallsProviderDestroyForTaintedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, taintedState)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1)
	testAssertMethodCalled(t, mp, "Create", 7) // ImageCache is always created
}

func TestApplyCallsProviderDestroyForDisabledResources(t *testing.T) {
	// resources that were created but subsequently set to disabled should be
	// destroyed
	e, mp := setupTestsWithState(t, nil, disabledAndCreatedState)

	_, err := e.Apply(context.Background(), "../../examples/disabled/config.hcl")
	require.NoError(t, err)

	// get the disabled resource
	r, err := e.config.FindResource("resource.container.consul_disabled")
	require.NoError(t, err)
	require.NotNil(t, r)

	// property should be disabled
	require.Equal(t, constants.StatusDisabled, r.Metadata().Properties[constants.PropertyStatus])

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 1, r)
	testAssertMethodCalled(t, mp, "Create", 0) // ImageCache is always created
}

func TestApplyCallsProviderRefreshForCreatedResources(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.NoError(t, err)

	// should only call twice as there is only one item in the state
	// that is in the config
	// should also be called for the image cache as there is a new network
	testAssertMethodCalled(t, mp, "Refresh", 2)
}

func TestApplyCallsProviderRefreshWithErrorHaltsExecution(t *testing.T) {
	e, mp := setupTestsWithState(t, map[string]error{"consul_config": fmt.Errorf("boom")}, singleFileState)

	_, err := e.Apply(context.Background(), "../../examples/single_file/container.hcl")
	require.Error(t, err)

	testAssertMethodCalled(t, mp, "Refresh", 3)
	testAssertMethodCalled(t, mp, "Create", 2)
}

func TestDestroyCallsProviderDestroyForEachProvider(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	err := e.Destroy(context.Background(), false)
	require.NoError(t, err)

	// should have call create for each provider
	// and once for the image cache
	testAssertMethodCalled(t, mp, "Destroy", 4)

	// state should be removed
	require.NoFileExists(t, utils.StatePath())
}

func TestDestroyCallsProviderDestroyForEachProviderWithEvents(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, existingState)

	events, err := e.Events()
	require.NoError(t, err)

	destroyingEventCount := 0
	destroyedEventCount := 0

	go func() {
		for event := range events {
			require.Contains(t, []constants.LifecycleEvent{constants.LifecycleEventDestroying, constants.LifecycleEventDestroyed}, event.Status)
			switch event.Status {
			case constants.LifecycleEventDestroying:
				destroyingEventCount += 1
			case constants.LifecycleEventDestroyed:
				destroyedEventCount += 1
			}
		}
	}()

	err = e.Destroy(context.Background(), false)
	require.NoError(t, err)

	// should have call create for each provider
	// and once for the image cache
	testAssertMethodCalled(t, mp, "Destroy", 4)

	// state should be removed
	require.NoFileExists(t, utils.StatePath())
	require.Equal(t, 4, destroyingEventCount)
	require.Equal(t, 4, destroyingEventCount)
	require.Empty(t, events)
}

func TestDestroyNotCallsProviderDestroyForResourcesDisabled(t *testing.T) {
	e, mp := setupTestsWithState(t, nil, disabledState)

	err := e.Destroy(context.Background(), false)
	require.NoError(t, err)

	// should have call create for each provider
	testAssertMethodCalled(t, mp, "Destroy", 2)
	testAssertMethodCalled(t, mp, "Create", 0) // ImageCache is always created

	// state should be removed
	require.NoFileExists(t, utils.StatePath())
}

func TestDestroyCallsProviderGenerateErrorStopsExecution(t *testing.T) {
	e, mp := setupTestsWithState(t, map[string]error{"mycontainer": fmt.Errorf("boom")}, complexState)

	err := e.Destroy(context.Background(), false)
	require.Error(t, err)

	// should have call destroy for each provider
	testAssertMethodCalled(t, mp, "Destroy", 2)

	// state should not be removed
	require.FileExists(t, utils.StatePath())
}

func TestDestroyFailSetsStatus(t *testing.T) {
	e, _ := setupTestsWithState(t, map[string]error{"mycontainer": fmt.Errorf("boom")}, complexState)

	err := e.Destroy(context.Background(), false)
	require.Error(t, err)

	r, _ := e.config.FindResource("resource.container.mycontainer")
	require.Equal(t, constants.StatusFailed, r.Metadata().Properties[constants.PropertyStatus])
}

func TestDestroyFailSetsStatusWithEvents(t *testing.T) {
	e, _ := setupTestsWithState(t, map[string]error{"mycontainer": fmt.Errorf("boom")}, complexState)

	events, err := e.Events()
	require.NoError(t, err)

	destroyingEventCount := 0
	destroyedEventCount := 0
	destroyFailedEventCount := 0

	go func() {
		for event := range events {
			require.Contains(t, []constants.LifecycleEvent{constants.LifecycleEventDestroying, constants.LifecycleEventDestroyed, constants.LifecycleEventDestroyFailed}, event.Status)
			switch event.Status {
			case constants.LifecycleEventDestroying:
				destroyingEventCount += 1
			case constants.LifecycleEventDestroyed:
				destroyedEventCount += 1
			case constants.LifecycleEventDestroyFailed:
				destroyFailedEventCount += 1
			}
		}
	}()

	err = e.Destroy(context.Background(), false)
	require.Error(t, err)

	r, _ := e.config.FindResource("resource.container.mycontainer")
	require.Equal(t, constants.StatusFailed, r.Metadata().Properties[constants.PropertyStatus])

	require.Equal(t, 2, destroyingEventCount)
	require.Equal(t, 1, destroyedEventCount)
	require.Equal(t, 1, destroyFailedEventCount)
	require.Empty(t, events)
}

func TestParseConfig(t *testing.T) {
	e, mp := setupTests(t, nil)

	r, err := e.ParseConfig("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, r.Resources, 6)

	// should not have created any providers
	testAssertMethodCalled(t, mp, "Events", 0)
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)
}

func TestParseConfigWithEvents(t *testing.T) {
	e, mp := setupTests(t, nil)

	events, err := e.Events()
	require.NoError(t, err)

	eventCount := 0
	go func() {
		for event := range events {
			eventCount += 1
			require.Equal(t, constants.LifecycleEventParsed, event.Status)
		}
	}()

	r, err := e.ParseConfig("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, r.Resources, 6)

	// should not have created any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)

	require.Equal(t, 6, eventCount)
	require.Empty(t, events)
}

func TestParseWithVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	r, err := e.ParseConfigWithVariables("../../examples/single_file/container.hcl", nil, "../../examples/single_file/default.vars")
	require.NoError(t, err)

	require.Len(t, r.Resources, 6)

	// should not have created any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)

	c, err := e.config.FindResource("resource.container.consul")
	require.NoError(t, err)
	require.Equal(t, "consul:1.8.1", c.(*container.Container).Image.Name)
}

func TestParseWithEnvironmentVariables(t *testing.T) {
	e, mp := setupTests(t, nil)

	err := os.Setenv("JUMPPAD_VAR_version", "consul:1.8.1")
	require.NoError(t, err)

	r, err := e.ParseConfig("../../examples/single_file/container.hcl")
	require.NoError(t, err)

	require.Len(t, r.Resources, 6)

	// should not have created any providers
	testAssertMethodCalled(t, mp, "Create", 0)
	testAssertMethodCalled(t, mp, "Destroy", 0)

	c, err := e.config.FindResource("resource.container.consul")
	require.NoError(t, err)
	require.Equal(t, "consul:1.8.1", c.(*container.Container).Image.Name)
}

func TestMultipleEventsCallsFail(t *testing.T) {
	e, _ := setupTests(t, nil)

	_, err := e.Events()
	require.NoError(t, err)

	_, err = e.Events()
	require.Error(t, err)
}

func testAssertMethodCalled(t *testing.T, p *mocks.Providers, method string, n int, resource ...types.Resource) {
	if len(resource) > 1 {
		panic("testAssertMethodCalled only expects 0 or 1 resources")
	}

	callCount := 0

	calls := map[string][]string{}

	for i, pm := range p.Providers {
		r := getResourceFromMock(p, i)
		// are we trying to filter on a specific resources
		if len(resource) == 1 {
			if resource[0] != r {
				continue
			}
		}

		for _, c := range pm.Calls {
			calls[r.Metadata().Name] = append(calls[r.Metadata().Name], c.Method)

			if c.Method == method {
				callCount++
			}
		}
	}

	callStrings := []string{}
	for k, v := range calls {
		calls := strings.Join(v, ",")
		callStrings = append(callStrings, fmt.Sprintf("%s[%s]", k, calls))
	}

	callString := strings.Join(callStrings, " ")

	require.Equal(t, n, callCount, fmt.Sprintf("expected %d calls, actual calls %d: %s", n, callCount, callString))
}

var failedState = `
{
  "resources": [
  {
      "meta": {
        "name": "onprem",
         "properties": {
          "status": "failed"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  }
  ]
}
`

var taintedState = `
{
  "resources": [
  {
      "meta": {
        "name": "onprem",
        "properties": {
          "status": "tainted"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  }
  ]
}
`

var existingState = `
{
  "resources": [
  {
      "meta": {
        "name": "cloud",
        "properties": {
          "status": "created"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  },
  {
      "meta": {
        "name": "default",
        "properties": {
          "status": "created"
        },
        "type": "image_cache"
      }
  },
  {
      "meta": {
        "name": "container",
        "properties": {
          "status": "created"
        },
        "type": "container"
      },
      "image": {
        "name": "test"
      }
  },
  {
      "meta": {
        "name": "consul_config",
        "properties": {
          "status": "created"
        },
        "type": "template"
      }
  }
  ]
}
`

var singleFileState = `
{
  "resources": [
  {
      "meta": {
        "name": "onprem",
        "properties": {
          "status": "created"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  },
  {
      "meta": {
        "name": "default",
        "properties": {
          "status": "created"
        },
        "type": "image_cache"
      }
  },
  {
      "meta": {
        "name": "consul",
        "properties": {
          "status": "created"
        },
        "type": "container"
      },
      "image": {
        "name": "test"
      }
  },
  {
      "meta": {
        "name": "consul_config",
        "properties": {
          "status": "created"
        },
        "type": "template"
      }
  },
  {
      "meta": {
        "name": "consul_addr",
        "properties": {
          "status": "created"
        },
        "type": "output"
      }
  }
  ]
}
`

var complexState = `
{
  "resources": [
  {
      "meta": {
        "name": "cloud",
        "properties": {
          "status": "created"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  },
  {
      "meta": {
        "name": "default",
        "type": "image_cache",
        "properties": {
          "status": "created"
        }
      },
      "depends_on": ["resource.network.cloud"]
  },
  {
      "meta": {
        "name": "mytemplate",
        "properties": {
          "status": "created"
        },
        "type": "template"
      }
  },
  {
      "meta": {
        "name": "mycontainer",
        "type": "container",
        "properties": {
          "status": "created"
        }
      },
      "depends_on": ["resource.network.cloud", "resource.template.mytemplate"]
  }
  ]
}
`

var disabledState = `
{
  "blueprint": null,
  "resources": [
  {
      "meta": {
        "name": "default",
        "type": "image_cache"
      }
  },
  {
      "meta": {
        "name": "dc1_enabled",
        "properties": {
          "status": "created"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  },
  {
     "disabled": true,
      "meta": {
        "name": "consul_disabled",
        "properties": {
           "status": "disabled"
         },
         "type": "container"
      }
  }
  ]
}
`

var disabledAndCreatedState = `
{
  "blueprint": null,
  "resources": [
  {
      "meta": {
        "name": "default",
        "properties": {
          "status": "created"
        },
        "type": "image_cache"
      }
  },
  {
      "meta": {
        "name": "dc1_enabled",
        "properties": {
          "status": "created"
        },
        "type": "network"
      },
      "subnet": "10.15.0.0/16"
  },
  {
     "disabled": false,
     "meta": {
        "name": "consul_disabled",
        "properties": {
          "status": "created"
        },
        "type": "container"
      },
      "image": {
        "name": "test"
      }	
  },
  {
     "disabled": false,
     "meta": {
      "name": "consul_enabled",
      "properties": {
        "status": "created"
      },
      "type": "container"
      },
      "image": {
        "name": "test"
      }
  }
  ]
}
`
