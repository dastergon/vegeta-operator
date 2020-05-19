package controller

import (
	"github.com/dastergon/vegeta-operator/pkg/controller/vegeta"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, vegeta.Add)
}
