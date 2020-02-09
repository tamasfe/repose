// This code was generated by Repose.

package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	api "github.com/tamasfe/repose/examples/echo-petstore/src/api"
)

type clientPets struct {
	server string
}

// PetsClient provides client requests for "/pets".
func PetsClient(server string) clientPets {
	return clientPets{server: server}
}

// AddPet provides client request for the operation.
func (c clientPets) AddPet(body api.NewPet) (*http.Request, error) {
	var _bodyData io.Reader
	var bodyData []byte
	if _b, err := json.Marshal(body); err != nil {
		return nil, err
	} else {
		bodyData = _b
	}
	_bodyData = bytes.NewBuffer(bodyData)

	url := c.server
	url += "/pets"

	_req, _err := http.NewRequest("POST", url, _bodyData)
	if _err != nil {
		return nil, _err
	}

	return _req, nil
}

// FindPets provides client request for the operation.
func (c clientPets) FindPets(limit int32, tags []string) (*http.Request, error) {
	var _bodyData io.Reader
	limitData := fmt.Sprint(limit)
	var _tagsArr []string
	for _, _p := range tags {
		_tagsArr = append(_tagsArr, fmt.Sprint(_p))
	}
	tagsData := strings.Join(_tagsArr, ",")

	url := c.server
	url += "/pets"

	_req, _err := http.NewRequest("GET", url, _bodyData)
	if _err != nil {
		return nil, _err
	}

	_req.URL.Query().Set("limit", string(limitData))
	_req.URL.Query().Set("tags", string(tagsData))

	return _req, nil
}

type clientPetsWithID struct {
	server string
}

// PetsWithIDClient provides client requests for "/pets/{id}".
func PetsWithIDClient(server string) clientPetsWithID {
	return clientPetsWithID{server: server}
}

// DeletePet provides client request for the operation.
func (c clientPetsWithID) DeletePet(id int64) (*http.Request, error) {
	var _bodyData io.Reader
	idData := fmt.Sprint(id)

	url := c.server
	url += "/pets/{id}"
	url = strings.Replace(url, "{id}", string(idData), 1)

	_req, _err := http.NewRequest("DELETE", url, _bodyData)
	if _err != nil {
		return nil, _err
	}

	return _req, nil
}

// FindPetById provides client request for the operation.
func (c clientPetsWithID) FindPetById(id int64) (*http.Request, error) {
	var _bodyData io.Reader
	idData := fmt.Sprint(id)

	url := c.server
	url += "/pets/{id}"
	url = strings.Replace(url, "{id}", string(idData), 1)

	_req, _err := http.NewRequest("GET", url, _bodyData)
	if _err != nil {
		return nil, _err
	}

	return _req, nil
}
