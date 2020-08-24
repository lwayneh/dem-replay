<<<<<<< HEAD
// Package common contains types that are used throughout this project.
package common

import (
	"time"

	"github.com/faiface/pixel"
	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	event "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
)

// Phase corresponds to a phase of a round.
type Phase int

// Possible values for Phase type.
const (
	PhaseFreezetime Phase = iota
	PhaseRegular
	PhasePlanted
	PhaseRestart
	PhaseWarmup
	PhaseHalftime
)

// OverviewState contains all information that will be displayed for a single tick.
type OverviewState struct {
	IngameTick            int
	Players               []Player
	Grenades              []common.GrenadeProjectile
	Infernos              []common.Inferno
	Bomb                  common.Bomb
	TeamCounterTerrorists Team
	TeamTerrorists        Team
	Timer                 Timer
	Health                map[string]int
}

//Player extends the Player type from the parser
type Player struct {
	common.Player
	Health         int
	ViewDirectionX float32
	Money          int
	Kills          int
	Deaths         int
	Assists        int
	Armor          int
	Helmet         bool
	Kit            bool
	ActiveWeapon   *common.Equipment
	FlashRemaining time.Duration
	ClanName       string
	ShortName      string
}

// Team extends the TeamState type from the parser
type Team struct {
	common.TeamState
	Score    int
	ClanName string
}

// Clan saves details of teams tags - used to shorten player names
type Clan struct {
	Tag      string
	ClanName string
}

// GrenadeEffect extends the GrenadeEvent type from the parser by the Lifetime
// variable that is used to draw the effect.
type GrenadeEffect struct {
	event.GrenadeEvent
	Lifetime int
}

// Kill contains all information that is displayed on the killfeed.
type Kill struct {
	KillerName string
	KillerTeam common.Team
	VictimName string
	VictimTeam common.Team
	Weapon     string
}

// Timer contains the time remaining in the current phase of the round.
type Timer struct {
	TimeRemaining time.Duration
	Phase         Phase
}

// Shot contains information about a shot from a weapon.
type Shot struct {
	Position       r3.Vector
	ViewDirectionX float32
	IsAwpShot      bool
}

// Control is an onscreen control for manipulating replay feedback (Play, Pause, Fastforward, Rewind, etc.)
type Control struct {
	Name   string
	Rect   pixel.Rect
	Offset pixel.Vec //Offset from center of control bar
	Scale  float64
	Sprite *pixel.Sprite
	Status Status
}

type Status struct {
	Name string
	Id   int
}

func (c *Control) SetScale(scale float64) {
	c.Scale = scale
}

func (c *Control) SetOffset(offset pixel.Vec) {
	c.Offset = offset
}
=======
// Package common contains types that are used throughout this project.
package common

import (
	"time"

	"github.com/golang/geo/r3"
	"github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/common"
	event "github.com/markus-wa/demoinfocs-golang/v2/pkg/demoinfocs/events"
)

// Phase corresponds to a phase of a round.
type Phase int

// Possible values for Phase type.
const (
	PhaseFreezetime Phase = iota
	PhaseRegular
	PhasePlanted
	PhaseRestart
	PhaseWarmup
	PhaseHalftime
)

// OverviewState contains all information that will be displayed for a single tick.
type OverviewState struct {
	IngameTick            int
	Players               []Player
	Grenades              []common.GrenadeProjectile
	Infernos              []common.Inferno
	Bomb                  common.Bomb
	TeamCounterTerrorists Team
	TeamTerrorists        Team
	Timer                 Timer
	Health                map[string]int
}

//Player extends the Player type from the parser
type Player struct {
	common.Player
	Health         int
	ViewDirectionX float32
	Money          int
	Kills          int
	Deaths         int
	Assists        int
	Armor          int
	Helmet         bool
	Kit            bool
}

// Team extends the TeamState type from the parser
type Team struct {
	common.TeamState
	Score    int
	ClanName string
}

// GrenadeEffect extends the GrenadeEvent type from the parser by the Lifetime
// variable that is used to draw the effect.
type GrenadeEffect struct {
	event.GrenadeEvent
	Lifetime int
}

// Kill contains all information that is displayed on the killfeed.
type Kill struct {
	KillerName string
	KillerTeam common.Team
	VictimName string
	VictimTeam common.Team
	Weapon     string
}

// Timer contains the time remaining in the current phase of the round.
type Timer struct {
	TimeRemaining time.Duration
	Phase         Phase
}

// Shot contains information about a shot from a weapon.
type Shot struct {
	Position       r3.Vector
	ViewDirectionX float32
	IsAwpShot      bool
}
>>>>>>> 3e09447e6ad137e8ff21ba225067df5bc4d25afc
