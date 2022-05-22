package js_test

import (
	"github.com/pflow-dev/go-metamodel/metamodel/js"
	"testing"
)

var testSrc = `
function TestModel (fn, cell, role) {
    r = role("default");
    console.log("DSL");
    p1 = cell("p1", 0, 1, { x: 1, y: 1, z: 0 });
    p2 = cell("p2", 0, 0, { x: 1, y: 2, z: 0 });
    p3 = cell("p3", 1, 1, { x: 1, y: 2, z: 0 });

    inc1 = fn("inc1", r, { x: 2, y: 1, z: 0 });
    inc1.tx(1, p1);

    dec1 = fn("dec1", r, { x: 2, y: 2, z: 0 });
    p1.tx(1, dec1);

    inc2 = fn("inc2", r, { x: 3, y: 1, z: 0 });
    inc2.tx(1, p2);

    dec2 = fn("dec2", r, { x: 3, y: 2, z: 0 });
    p2.tx(1, dec2);

    inc3 = fn("inc3", r, { x: 4, y: 1, z: 0 });
    inc3.tx(1, p3);

    dec3 = fn("dec3", r, { x: 4, y: 2, z: 0 });
    p3.tx(1, dec3);

    p3.guard(1, inc1);
}


m = domodel("Test", TestModel);
/*
s = m.initialVector();

function resolve(res) {
    console.log("OK "+JSON.stringify(res));
}

function reject(res) {
    console.log("FAIL "+JSON.stringify(res));
}

m.fire(s, "inc2", 3, resolve, reject);
console.log({m: m})

// out = m.initialVector()
console.log(JSON.stringify(s)+"<= state");
console.log(JSON.stringify(m.def.transitions['inc1'])+"<= model");
*/
`

func TestJsLoader(t *testing.T) {
	m, _ := js.LoadModel(testSrc)
	v := m.InitialVector()
	t.Logf("%v", v)
}
