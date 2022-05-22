package examples

import "github.com/pflow-dev/go-metamodel/metamodel"

type Quest struct {
	metamodel.MetaModel
}

func NewQuest() *Quest {
	mm := metamodel.New("Quest")
	mm.Define(quest)
	return &Quest{mm}
}

func quest(m metamodel.Declaration) {
	cell, fn := m.Cell, m.Fn

	player := "player"
	// admin := "admin"

	permission := func(label string) metamodel.Node {
		action := fn().Label(label).Role(player)
		hold := cell().Label(label).Initial(1).Capacity(1)
		hold.Guard(1, action)
		return hold
	}

	permission("summon_tardis")
	permission("noclip")
	permission("board")

	fly := permission("fly")
	unlockFly := fn().Label("unlock_fly").Role(player)
	fly.Tx(1, unlockFly)

	waypoint := func(label string, pos metamodel.Position) metamodel.Node {
		pin := fn().Label("pin_"+label).Role(player).Position(pos.X, pos.Y, pos.Z)
		mark := cell().Label(label).Capacity(1).Position(pos.X, pos.Y, pos.Z)
		pin.Tx(1, mark)
		mark.Tx(1, unlockFly) // visiting all waypoints unlocks fly
		return pin
	}

	spaceport := waypoint("spaceport", metamodel.Position{X: 0, Y: 0, Z: 0})
	orbital := waypoint("orbital", metamodel.Position{X: 0, Y: 0, Z: 0})

	ship := func(label string, coords metamodel.Position) metamodel.Node {
		board_ship := fn().Label("board_" + label)
		passengers := cell().Label("aboard_" + label)
		_ = passengers
		_ = board_ship
		return board_ship
	}

	tardis := waypoint("tardis", metamodel.Position{X: 0, Y: 0, Z: 0})
	_ = ship
	_ = orbital
	_ = tardis
	_ = spaceport
	_ = cell
	_ = fn
}
