package ops

import "github.com/mariexaos/wepod/internal/domain"

// CreateProgressEvent is emitted at each step of a Create operation.
type CreateProgressEvent struct {
	ID    domain.InstanceID
	Step  int
	Total int
	Label string
}

// UpdateProgressEvent is emitted at each step of an Update operation.
type UpdateProgressEvent struct {
	ID    domain.InstanceID
	Step  int
	Total int
	Label string
}

// DeleteProgressEvent is emitted as each copy is deleted.
type DeleteProgressEvent struct {
	ID       domain.InstanceID
	Done     int
	Total    int
	WithData bool
}

// IconApplyEvent is emitted as the icon is applied to each copy.
type IconApplyEvent struct {
	ID       domain.InstanceID
	Done     int
	Total    int
	IconName string
}
