package edition

import "sync/atomic"

type Edition struct {
	Name         string
	MaxAgents    int // 0 = unlimited
	MaxTeams     int
	HasRBAC      bool
	HasSync      bool
	HasAdmin     bool
	HasBilling   bool
	HasScheduler bool
	DBBackend    string // "sqlite" or "postgresql"
}

// IsLimited returns true if the edition has a cap on the number of agents.
func (e Edition) IsLimited() bool {
	return e.MaxAgents > 0
}

// AgentLimitReached returns true when the current agent count meets or exceeds the limit.
// Always returns false for unlimited editions.
func (e Edition) AgentLimitReached(current int) bool {
	if e.MaxAgents == 0 {
		return false
	}
	return current >= e.MaxAgents
}

func Lite() Edition {
	return Edition{
		Name: "Lite", MaxAgents: 5, MaxTeams: 1,
		HasRBAC: false, HasSync: false, HasAdmin: false, HasBilling: false,
		HasScheduler: false, DBBackend: "sqlite",
	}
}

func Enterprise() Edition {
	return Edition{
		Name: "Enterprise", MaxAgents: 0, MaxTeams: 0,
		HasRBAC: true, HasSync: true, HasAdmin: true, HasBilling: true,
		HasScheduler: true, DBBackend: "postgresql",
	}
}

var current atomic.Pointer[Edition]

func init() {
	e := Lite()
	current.Store(&e)
}

func Current() Edition     { return *current.Load() }
func SetEdition(e Edition) { current.Store(&e) }
