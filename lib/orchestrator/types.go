package orchestrator

import (
	"github.com/sid-technologies/pilum/lib/recepie"
	serviceinfo "github.com/sid-technologies/pilum/lib/service_info"
)

// stepTask represents a task for a specific service at a specific step.
type stepTask struct {
	service serviceinfo.ServiceInfo
	recipe  recepie.Recipe
	step    *recepie.RecipeStep
}
