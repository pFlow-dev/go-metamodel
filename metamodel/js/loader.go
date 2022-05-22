package js

import (
	"encoding/json"
	"github.com/pflow-dev/go-metamodel/metamodel"
	"github.com/robertkrimen/otto"
)

func LoadModel(src string) (*metamodel.Model, error) {

	vm := otto.New()
	_, err := vm.Run(internalDsl)
	if err != nil {
		panic(err)
	}
	_, err = vm.Run(src)
	if err != nil {
		panic(err)
	}

	pnet := new(metamodel.PetriNet)
	if value, err := vm.Get("Model"); err == nil {
		data, _ := value.ToString()
		err = json.Unmarshal([]byte(data), pnet)
		if err != nil {
			panic(err)
		}
	}
	model := &metamodel.Model{PetriNet: pnet}
	model.Graph().Index() // rebuild and reindex
	return model, nil
}
