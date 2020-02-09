// This code generated by Repose.

package server

import (
	"fmt"

	echo "github.com/labstack/echo/v4"
	api "github.com/tamasfe/repose/examples/echo-petstore/src/api"
)

// repose:keep server_def
// The struct used for implementing Server.
// Repose relies on the name, make sure to keep it updated in its config.
type ServerImpl struct{}

// repose:endkeep

// Make sure that we implement the correct server.
var _ api.Server = &ServerImpl{}

// AddPet is the "POST" operation for path "/pets".
//
// Description: Creates a new pet in the store. Duplicates are allowed.
//
// Parameters:
//     "body" in body with content-type application/json.
//     Description: Pet to add to the store.
//
// Responses:
//     "Pet" (200): with content-type application/json.
//     Description: pet response.
//
//     "Error" (default): with content-type application/json.
//     Description: unexpected error.
//
func (s *ServerImpl) AddPet(c echo.Context, body *api.NewPet) (api.AddPetHandlerResponse, error) {
	// repose:keep AddPet_body
	return &api.Pet{
		NewPet: *body,
		PetFragment1: api.PetFragment1{
			ID: 3,
		},
	}, nil
	// repose:endkeep
}

// FindPets is the "GET" operation for path "/pets".
//
// Description: Returns all pets from the system that the user has access to
// Nam sed condimentum est. Maecenas tempor sagittis sapien, nec rhoncus sem
// sagittis sit amet. Aenean at gravida augue, ac iaculis sem. Curabitur odio
// lorem, ornare eget elementum nec, cursus id lectus. Duis mi turpis, pulvinar ac
// eros ac, tincidunt varius justo. In hac habitasse platea dictumst. Integer at
// adipiscing ante, a sagittis ligula. Aenean pharetra tempor ante molestie
// imperdiet. Vivamus id aliquam diam. Cras quis velit non tortor eleifend
// sagittis. Praesent at enim pharetra urna volutpat venenatis eget eget mauris. In
// eleifend fermentum facilisis. Praesent enim enim, gravida ac sodales sed,
// placerat id erat. Suspendisse lacus dolor, consectetur non augue vel, vehicula
// interdum libero. Morbi euismod sagittis libero sed lacinia.
// Sed tempus felis lobortis leo pulvinar rutrum. Nam mattis velit nisl, eu
// condimentum ligula luctus nec. Phasellus semper velit eget aliquet faucibus. In
// a mattis elit. Phasellus vel urna viverra, condimentum lorem id, rhoncus nibh.
// Ut pellentesque posuere elementum. Sed a varius odio. Morbi rhoncus ligula
// libero, vel eleifend nunc tristique vitae. Fusce et sem dui. Aenean nec
// scelerisque tortor. Fusce malesuada accumsan magna vel tempus. Quisque mollis
// felis eu dolor tristique, sit amet auctor felis gravida. Sed libero lorem,
// molestie sed nisl in, accumsan tempor nisi. Fusce sollicitudin massa ut lacinia
// mattis. Sed vel eleifend lorem. Pellentesque vitae felis pretium, pulvinar elit
// eu, euismod sapien.
//
// Parameters:
//     "limit" in query.
//     Description: maximum number of results to return.
//
//     "tags" in query.
//     Description: tags to filter by.
//
// Responses:
//     "FindPetsResponse200" (200): with content-type application/json.
//     Description: pet response.
//
//     "Error" (default): with content-type application/json.
//     Description: unexpected error.
//
func (s *ServerImpl) FindPets(c echo.Context, limit *int32, tags []string) (api.FindPetsHandlerResponse, error) {
	// repose:keep FindPets_body
	return nil, fmt.Errorf("This is not implemented")
	// repose:endkeep
}

// DeletePet is the "DELETE" operation for path "/pets/{id}".
//
// Description: deletes a single pet based on the ID supplied.
//
// Parameters:
//     "id" in path.
//     Description: ID of pet to delete.
//
// Responses:
//     "DeletePetResponse204" (204).
//     Description: pet deleted.
//
//     "Error" (default): with content-type application/json.
//     Description: unexpected error.
//
func (s *ServerImpl) DeletePet(c echo.Context, id int64) (api.DeletePetHandlerResponse, error) {
	// repose:keep DeletePet_body
	return api.DeletePetResponse204, nil
	// repose:endkeep
}

// FindPetById is the "GET" operation for path "/pets/{id}".
//
// Description: Returns a user based on a single ID, if the user does not have
// access to the pet.
//
// Parameters:
//     "id" in path.
//     Description: ID of pet to fetch.
//
// Responses:
//     "Pet" (200): with content-type application/json.
//     Description: pet response.
//
//     "Error" (default): with content-type application/json.
//     Description: unexpected error.
//
func (s *ServerImpl) FindPetById(c echo.Context, id int64) (api.FindPetByIdHandlerResponse, error) {
	// repose:keep FindPetById_body
	return &api.Pet{
		NewPet: api.NewPet{
			Name: "Bonnie",
		},
		PetFragment1: api.PetFragment1{
			ID: id,
		},
	}, nil
	// repose:endkeep
}

// Middleware allows attaching middleware to each operation.
func (s *ServerImpl) Middleware() *api.ServerMiddleware {
	// repose:keep middleware_body
	return nil
	// repose:endkeep
}