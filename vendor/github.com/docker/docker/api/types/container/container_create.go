package container // import "github.com/docker/docker/api/types/container"

// ----------------------------------------------------------------------------
// Code generated by `swagger generate operation`. DO NOT EDIT.
//
// See hack/generate-swagger-api.sh
// ----------------------------------------------------------------------------

// ContainerCreateCreatedBody OK response to ContainerCreate operation
// swagger:model ContainerCreateCreatedBody
type ContainerCreateCreatedBody struct {

	// The ID of the created container
	// Required: true
	ID string `json:"Id"`

	// Warnings encountered when creating the container
	// Required: true
	Warnings []string `json:"Warnings"`
}
