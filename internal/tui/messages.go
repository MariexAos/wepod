package tui

import (
	"time"

	"github.com/mariexaos/wepod/internal/domain"
	"github.com/mariexaos/wepod/internal/ops"
)

// Inbound tea.Msg types. Kept package-private; only Update consumes them.

type instancesLoadedMsg struct {
	Items []domain.Instance
	Err   error
}

type runtimeRefreshedMsg struct {
	Status map[domain.InstanceID]domain.Runtime
}

type createProgressMsg ops.CreateProgressEvent
type createDoneMsg struct {
	ID  domain.InstanceID
	Err error
}

type deleteProgressMsg ops.DeleteProgressEvent
type deleteDoneMsg struct {
	IDs []domain.InstanceID
	Err error
}

type updateProgressMsg ops.UpdateProgressEvent
type updateDoneMsg struct {
	IDs []domain.InstanceID
	Err error
}

type launchDoneMsg struct {
	ID  domain.InstanceID
	Err error
}

type stopDoneMsg struct {
	Err error
}

type iconProgressMsg ops.IconApplyEvent
type iconAppliedMsg struct {
	IDs []domain.InstanceID
	Err error
}

type iconsLoadedMsg struct {
	Paths []string
	Err   error
}

type sudoRefreshedMsg struct {
	Err error
}

type runtimeTickMsg time.Time
type errMsg struct {
	Op  string
	Err error
}
