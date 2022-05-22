package lua_test

import (
	"github.com/pflow-dev/go-metamodel/metamodel"
	"github.com/pflow-dev/go-metamodel/metamodel/lua"
	"testing"
)

func TestModel_TicTacToe(t *testing.T) {
	m, err := lua.LoadModel(`require "metamodel"
		domodel("TicTacToe", function (fn, cell, role)

		    function row(n)
		        return {
		            [0] = cell(n..0, 1, 1),
		            [1] = cell(n..1, 1, 1),
		            [2] = cell(n..2, 1, 1)
		        }
		    end

		    local board = {
		        [0] = row(0),
		        [1] = row(1),
		        [2] = row(2)
		    }

		    local X, O = "X", "O"

		    local players = {
		        [X] = {
		            turn = cell(X, 1, 1),
		            role = role(X),
		            next = O
		        },
		        [O] = {
		            turn = cell(O, 0, 1),
		            role = role(O),
		            next = X
		        }
		    }

		    for i, board_row in pairs(board) do
		        for j in pairs(board_row) do
		            for marking, player in pairs(players) do
		                local move = fn(marking..i..j, player.role)
		                player.turn.tx(1, move)
		                board[i][j].tx(1, move)
		                move.tx(1, players[player.next].turn)
		            end
		        end
		    end
		end)
	`)

	if err != nil || m.Schema != "TicTacToe" {
		t.Fatalf("failed to load TicTacToe %s", err)
	}

	if m == nil {
		t.Fatal("failed to load model")
	}
	if len(m.Places) != 11 {
		t.Fatal("expected 11 places")
	}

	initial := m.InitialVector()

	if initial[m.Places["X"].Offset] != 1 {
		t.Fatalf("Failed to init X move first")
	}

	if initial[m.Places["O"].Offset] != 0 {
		t.Fatalf("Failed to init O move second")
	}

	ps := m.Execute()

	move := func(action string) bool {
		ok, msg, out := ps.Fire(metamodel.Op{Action: action, Multiple: 1, Role: action[:1]})
		t.Logf("move: %v %v %v \n", ok, msg, out)
		return ok
	}

	assertOk := func(action string) {
		if !move(action) {
			t.Fatal("expected action to succeed")
		}
	}
	assertFail := func(action string) {
		if move(action) {
			t.Fatal("expected action to succeed")
		}
	}

	assertFail("O11")
	assertOk("X11")
	assertOk("O01")
	assertFail("X11")
}

func TestModel_Counter(t *testing.T) {
	m, err := lua.LoadModel(`
		domodel("Counter", function (fn, cell, role)
			local user = role("user")

			local p0 = cell('p0', 1, 0, {x=1,y=0})
			local inc0 = fn('inc0', user, {x=1, y=1})
			inc0.tx(1, p0)
			local dec0 = fn('dec0', user, {x=1, y=1})
			p0.tx(1, dec0)

			local p1 = cell('p1', 0, 1, {x=1,y=1})
			local inc1 = fn('inc1', user, {x=1, y=2})
			inc1.tx(1, p1)
			local dec1 = fn('dec0', user, {x=1, y=1})
			p1.tx(1, dec1)

			local p2 = cell('p2', 0, 1, {x=1,y=3})
			p2.guard(1, inc0)
			local inc2 = fn('inc2', user, {x=1, y=3})
			inc2.tx(1, p2)
			dec2 = fn('dec2', user)
			p2.tx(1, dec2)
		end)
	`)

	if err != nil {
		t.Fatalf("failed to load counter %s", err)
	}

	assertEq := func(a string, b string, message ...string) {
		msg := ""
		if len(message) == 1 {
			msg = message[0]
		}
		if a != b {
			t.Fatalf("%v != %v %s", a, b, msg)
		}
	}

	assertEq(m.Places["p1"].Label, "p1")
	ps := m.Execute()
	t.Logf("state: %v\n", ps.GetState())

	move := func(action string, multiple int64) bool {
		ok, msg, out := ps.Fire(metamodel.Op{Action: action, Multiple: multiple, Role: "user"})
		t.Logf("move: %v %v %v \n", ok, msg, out)
		return ok
	}

	assertOk := func(action string, multiple int64) {
		if !move(action, multiple) {
			t.Fatal("expected action to succeed")
		}
	}
	assertFail := func(action string, multiple int64) {
		if move(action, multiple) {
			t.Fatal("expected action to succeed")
		}
	}

	assertOk("inc0", 1)
	assertFail("inc1", 3)
	assertOk("inc2", 1)
	assertFail("inc0", 1)
	assertOk("dec2", 1)
	assertOk("inc0", 1)
}

func TestModel_Metasyntax(t *testing.T) {
	m, err := lua.LoadModel(`
		domodel("metasyntax", function (fn, cell, role)
			local defaultRole =role("default")

			local foo =cell("foo", 1, 0, {x=170, y=230})
			local baz =cell("baz", 0, 0, {x=330, y=110})

			local bar =fn("bar", defaultRole, {x=170, y=110})
			local qux =fn("qux", defaultRole, {x=330, y=230})
			local quux =fn("quux", defaultRole, {x=50, y=230})
			local plugh =fn("plugh", defaultRole, {x=460, y=110})

			foo.tx(1, bar)
			qux.tx(1, foo)

			baz.tx(1, qux)
			bar.tx(1, baz)

			foo.guard(1, quux)
			baz.guard(1, plugh)
		end)
	`)

	_ = err
	tx := m.PetriNet.Transitions["quux"]
	if tx.Guards["foo"].Label != "foo" {
		t.Fatalf("Failed to find guard foo")
	}
}
