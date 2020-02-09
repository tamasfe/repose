package util

import (
	"testing"

	"gopkg.in/go-playground/assert.v1"
)

func TestConvertParams(t *testing.T) {
	origStr := "/asd/{bsd}/some/thing/with/{id}/{and}/{random}/parameters"

	expectedColonStr := "/asd/:bsd/some/thing/with/:id/:and/:random/parameters"

	colonStr := ParamStyleToColon(origStr)

	assert.Equal(t, colonStr, expectedColonStr)

	bracesStr := ParamStyleToBraces(colonStr)

	assert.Equal(t, bracesStr, origStr)
}
