package ops

import "github.com/mariexaos/wepod/internal/domain"

// SetIconStash overrides the icon-stash path used by Update and returns a
// function that restores the previous behavior. Test-only.
func SetIconStash(fn func(domain.Instance) string) (restore func()) {
	prev := iconStash
	iconStash = fn
	return func() { iconStash = prev }
}
