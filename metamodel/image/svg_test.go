package image_test

import (
	"github.com/pflow-dev/go-metamodel/metamodel/image"
	"github.com/pflow-dev/go-metamodel/metamodel/lua"
	"testing"
)

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
	i := image.NewSvgFile("/tmp/test.svg", 512, 256)
	i.Render(m)
}
