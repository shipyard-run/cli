package constants

// PropertyStatus is the key for the Metadata property that contains the status
const PropertyStatus = "status"

const (
	// StatusCreated is set once the resource has been successfully created
	StatusCreated = "created"

	// StatusTainted indicates that the resources has been successfully created
	// but should be destroyed and re-created
	StatusTainted = "tainted"

	// StatusFailed indicates that the resource failed to create
	StatusFailed = "failed"

	// StatusDisabled indicates that the resources has been disabled and no
	// resources have been created
	StatusDisabled = "disabled"
)

type LifecycleEvent string

const (
	// LifecycleEventParsed is sent once the resource has been successfully parsed
	LifecycleEventParsed LifecycleEvent = "parsed"

	// LifecycleEventCreating is sent once the resource is about to be created
	LifecycleEventCreating LifecycleEvent = "creating"

	// LifecycleEventCreated is sent once the resource has been successfully created
	LifecycleEventCreated LifecycleEvent = "created"

	// LifecycleEventCreatedFailed is sent once the resource has failed to create
	LifecycleEventCreatedFailed LifecycleEvent = "create_failed"

	// LifecycleEventDestroying is sent once the resource is about to be destroyed
	LifecycleEventDestroying LifecycleEvent = "destroying"

	// LifecycleEventDestroyed is sent once the resource has been destroyed
	LifecycleEventDestroyed LifecycleEvent = "destroyed"

	// LifecycleEventDestroyFailed is sent once the resource has failed to create
	LifecycleEventDestroyFailed LifecycleEvent = "destroy_failed"
)
