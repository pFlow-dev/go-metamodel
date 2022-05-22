package lua

import (
	"errors"
	"github.com/pflow-dev/go-metamodel/metamodel"
	"github.com/yuin/gluamapper"
	. "github.com/yuin/gopher-lua"
	"regexp"
	"strings"
)

var L *LState
var Models map[string]*metamodel.PetriNet

func init() {
	L = NewState()
	err := L.DoString(internalDsl)
	if err != nil {
		panic(err)
	}
}

// LoadModel evaluates lua source code and maps models into go
func LoadModel(modelSource string) (model *metamodel.Model, err error) {
	modelName := regexp.MustCompile(`domodel\(["'](\w+)["']`)
	out := modelName.FindStringSubmatch(modelSource)
	if len(out) != 2 {
		return nil, errors.New(`missing: domodel("<schema>"`)
	}
	schema := out[1]
	if err != nil {
		return nil, err
	}
	src := strings.Replace(modelSource, "require ", "-- require ", -1) // comment out require statements
	// fmt.Printf("%v", src)
	err = L.DoString(src)
	if err != nil {
		return nil, err
	}
	if err = gluamapper.Map(L.GetGlobal("Models").(*LTable), &Models); err != nil {
		return nil, err
	}

	key := ""
	for k, _ := range Models { // KLUDGE
		if strings.ToLower(k) == strings.ToLower(schema) {
			key = k
			break
		}
	}
	pnet, ok := Models[key]
	if !ok {
		return nil, errors.New("Failed to load model: " + schema)
	}
	delete(Models, schema) // purge cache
	model = &metamodel.Model{PetriNet: pnet}

	{ // Fix side effects of using gluamapper
		for i, p := range model.Places {
			p.Offset = p.Offset - 1 // Lua uses 1-indexed arrays Go is 0-indexed
			delete(pnet.Places, i)
			pnet.Places[p.Label] = p // overwrite keys w/ label (gluamapper up-cases)
		}
		for i, t := range pnet.Transitions {
			delete(pnet.Transitions, i)
			pnet.Transitions[t.Label] = t // overwrite keys w/ label (gluamapper up-cases)

			for j, g := range t.Guards { // correct caps w/ guards
				delete(t.Guards, j)
				t.Guards[g.Label] = g
			}
		}
		pnet.Schema = schema
	}

	model.Graph().Index() // rebuild arcs and reindex
	return model, err
}
