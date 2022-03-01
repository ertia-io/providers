package dependencies

import (
	ertia "github.com/ertia-io/config/pkg/entities"
)

var K3SDependency = ertia.Dependency{
	Name:    "K3S",
	Status:  ertia.DependencyStatusNew,
	Retries: 0,
}
