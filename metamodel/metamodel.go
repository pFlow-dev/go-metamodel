package metamodel

import (
	"archive/zip"
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

var x = json.Unmarshal

// Position defines location of a Place or Transition element
type Position struct {
	X int64 `json:"x"`
	Y int64 `json:"y"`
	Z int64 `json:"z"`
}

type Label = string

// Role defines user permission
type Role = struct {
	Label `json:"label"`
}

type RoleMap = map[string]Role

type Vector = []int64

func VectorToBytes(v Vector) []byte {
	bv := make([]byte, len(v))
	for i, b := range v {
		bv[i] = byte(b)
	}
	return bv
}

func VectorFromBytes(bv []byte) (v Vector) {
	v = make([]int64, len(bv))
	for i, b := range bv {
		byteToInt := int(b)
		if byteToInt > 127 {
			v[i] = int64(byteToInt - 256)
		} else {
			v[i] = int64(byteToInt)
		}
	}
	return v
}

// Place elements contain tokens
type Place struct {
	Label    string `json:"label"`
	Offset   int64  `json:"offset"`
	Position `json:"position"`
	Initial  int64 `json:"initial"`
	Capacity int64 `json:"capacity"`
}

type PlaceMap = map[string]*Place

// Guard attributes inhibit a transition
type Guard struct {
	Label    string `json:"label"`
	Delta    Vector `json:"delta"`
	Inverted bool   `json:"inverted"`
}

type GuardMap = map[string]*Guard

type SubnetNode struct {
	*PetriNet  `json:"-"`
	SubnetType string `json:"subnetType"`
	Schema     string `json:"schema"`
}

// Transition defines a token transfer action
type Transition struct {
	Label        string `json:"label"`
	Position     `json:"position"`
	Role         Role     `json:"role"`
	Delta        Vector   `json:"delta"`
	Guards       GuardMap `json:"guards"`
	AllowReentry bool     `json:"allowReentry"`
	*SubnetNode  `json:"subNet"`
}

type TransitionMap = map[string]*Transition

// Node is an interstitial interface used when composing m elements
type Node interface {
	Tx(weight int64, target Node) Node
	Guard(weight int64, target Node) Node
	IsPlace() bool
	IsTransition() bool
	GetPlace() *Place
	GetTransition() *Transition
	Label(string) Node
	Position(x int64, y int64, z ...int64) Node
	Initial(int64) Node
	Capacity(int64) Node
	Role(string) Node
}

// Add vectors while asserting underflow & capacity checks
func Add(state Vector, delta Vector, multiple int64, capacity ...Vector) (ok bool, msg string, out Vector) {
	out = make([]int64, len(state))
	if multiple <= 0 {
		return ok, msg, out
	} else {
		ok = true
	}
	if len(capacity) == 0 {
		for i, v := range state {
			out[i] = v + delta[i]*multiple
			if out[i] < 0 {
				msg = Underflow
				ok = false
			}
		}
	} else {
		for i, v := range state {
			out[i] = v + delta[i]*multiple
			if out[i] < 0 {
				msg = Underflow
				ok = false
			} else if capacity[0][i] > 0 && out[i] > capacity[0][i] {
				msg = Overflow
				ok = false
			}
		}
	}
	return ok, msg, out
}

type node struct {
	m *Model
	*Place
	*Transition
}

// Initial set the initial token value
func (n *node) Initial(i int64) Node {
	if n.IsPlace() {
		n.Place.Initial = i
	} else {
		panic(ExpectedPlace)
	}
	return n
}

// Capacity sets max tokens a place can store 0 = unlimited
func (n *node) Capacity(i int64) Node {
	if n.IsPlace() {
		n.Place.Capacity = i
	} else {
		panic(ExpectedPlace)
	}
	return n
}

// Tx defines a path between elements
func (n *node) Tx(weight int64, target Node) Node {
	if n.IsPlace() && target.IsPlace() {
		panic(BadArcPlace)
	}
	if n.IsTransition() && target.IsTransition() {
		panic(BadArcTransition)
	}
	if weight < 0 {
		panic(BadWeight)
	}
	n.m.Arcs = append(n.m.Arcs, Arc{
		Source:    n,
		Target:    target,
		Weight:    weight,
		Inhibitor: false,
	})
	return n
}

// Guard defines an inhibitor rule
func (n *node) Guard(weight int64, target Node) Node {
	var inverted = false
	if n.IsTransition() {
		if !target.IsPlace() {
			panic(BadInhibitorTarget)
		}
		inverted = true
	}
	if target.IsPlace() {
		if !n.IsTransition() {
			panic(BadInhibitorTarget)
		}
	}
	if weight < 0 {
		panic(BadWeight)
	}
	n.m.Arcs = append(n.m.Arcs, Arc{
		Source:    n,
		Target:    target,
		Weight:    weight,
		Inhibitor: true,
		Inverted:  inverted,
	})
	return n
}

func (n *node) IsPlace() bool {
	return n.Transition == nil
}

func (n *node) IsTransition() bool {
	return n.Place == nil
}

func (n *node) GetPlace() *Place {
	return n.Place
}

func (n *node) GetTransition() *Transition {
	return n.Transition
}

// Position sets the graphical position of an element
func (n *node) Position(x int64, y int64, z ...int64) Node {
	zee := int64(0)
	if len(z) == 1 {
		zee = z[0]
	}
	if n.IsPlace() {
		n.Place.Position = Position{x, y, zee}
	} else if n.IsTransition() {
		n.Transition.Position = Position{x, y, zee}
	}
	return n
}

// Label sets the name of an element
func (n *node) Label(label string) Node {
	if n.IsPlace() {
		n.m.Places[label] = n.Place
		delete(n.m.Places, n.Place.Label)
		n.Place.Label = label
	} else if n.IsTransition() {
		n.m.Transitions[label] = n.Transition
		delete(n.m.Transitions, n.Transition.Label)
		n.Transition.Label = label
	}
	return n
}

// Role sets the role of an element
func (n *node) Role(label string) Node {
	if n.IsTransition() {
		r := Role{Label: label}
		n.m.Roles[label] = r
		n.Transition.Role = r
	} else {
		panic(ExpectedTransition)
	}
	return n
}

type Arc struct {
	Source    Node
	Target    Node
	Weight    int64
	Inhibitor bool
	Inverted  bool
}

type PetriNet struct {
	Schema      string        `json:"schema"`
	Places      PlaceMap      `json:"places"`
	Transitions TransitionMap `json:"transitions"`
	Arcs        []Arc         `json:"-"`
	Roles       RoleMap       `json:"-"`
	Path        string        `json:"path"`
	Cid         string        `json:"cid"`
}

func (n *PetriNet) EmptyVector() (v Vector) {
	return make([]int64, len(n.Places))
}

func (n *PetriNet) InitialVector() (v Vector) {
	v = n.EmptyVector()
	for _, p := range n.Places {
		v[p.Offset] = p.Initial
	}
	return v
}

func (n *PetriNet) CapacityVector() (v Vector) {
	v = n.EmptyVector()
	for _, p := range n.Places {
		v[p.Offset] = p.Capacity
	}
	return v
}

const (
	BadInhibitorSource  = "inhibitor source must be a place"
	BadInhibitorTarget  = "inhibitor target must be a transition"
	BadWeight           = "arc weight must be positive integer"
	BadMultiple         = "multiple must be positive integer"
	BadArcTransition    = "source and target are both transitions"
	BadArcPlace         = "source and target are both places"
	UnknownAction       = "unknown action"
	Underflow           = "output cannot be negative"
	Overflow            = "output exceeds capacity"
	FailedRoleAssertion = "role assertion failed"
	ExpectedTransition  = "element was expected to be a transition"
	ExpectedPlace       = "element was expected to be a place"
	InhibitedTransition = "transition is inhibited by place %s"
	UnexpectedArguments = "expected %v arguments got %v"
	OK                  = "OK"
	defaultMultiple     = 1
)

type Op struct {
	Action   string
	Multiple int64
	Role     string
}

type Event struct {
	Seq   int64
	State Vector
	Op
}

type PlaceDefinition struct {
	Initial  int64 `json:"initial"`
	Capacity int64 `json:"capacity"`
	X        int64 `json:"x"`
	Y        int64 `json:"y"`
}
type TransitionDefinition struct {
	Role string `json:"role"`
	X    int64  `json:"x"`
	Y    int64  `json:"y"`
}
type ArcDefinition struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Weight  int64  `json:"weight"`
	Inhibit bool   `json:"inhibit"`
}

type PlaceMapDefinition map[string]PlaceDefinition
type TransitionMapDefinition map[string]TransitionDefinition
type ArcListDefinition []ArcDefinition

type DeclarationObject struct {
	ModelType   string                  `json:"modelType"`
	Version     string                  `json:"version"`
	Places      PlaceMapDefinition      `json:"places"`
	Transitions TransitionMapDefinition `json:"transitions"`
	Arcs        ArcListDefinition       `json:"arcs"`
}

type Process interface {
	GetState() Vector
	TokenCount(string) int64
	Inhibited(Op) (flag bool, label string)
	TestFire(Op) (flag bool, msg string, out Vector)
	Fire(Op) (ok bool, msg string, out Vector)
}

// REVIEW: should we expose Role here also?
type Declaration interface {
	Cell(...func(p *Place)) Node
	Fn(...func(t *Transition)) Node
}

type Editor interface {
	PlaceSeq() Label
	TransitionSeq() Label
	Index() Editor
	Graph() Editor
}

type MetaModel interface {
	Net() *PetriNet
	Define(...func(Declaration)) MetaModel
	Execute(...Vector) Process
	Edit() Editor
	Node(oid string) Node
	UnzipUrl(url string) (json string, ok bool)
	GetSize() (width int, height int)
}

type Model struct {
	*PetriNet
}

func (m *Model) GetSize() (width int, height int) {
	var limitX int64 = 0
	var limitY int64 = 0

	for _, p := range m.Places {
		if limitX < p.X {
			limitX = p.X
		}
		if limitY < p.Y {
			limitY = p.Y
		}
	}
	for _, t := range m.Transitions {
		if limitX < t.X {
			limitX = t.X
		}
		if limitY < t.Y {
			limitY = t.Y
		}
	}
	const margin = 60

	if width == 0 {
		width = int(limitX) + margin
	}
	if height == 0 {
		height = int(limitY) + margin
	}
	return width, height
}

func (m *Model) loadJsonDefinition(obj string) (ok bool) {
	ok = false
	if obj == "" {
		return false
	}
	modelObject := DeclarationObject{}
	err := json.Unmarshal([]byte(obj), &modelObject)
	if err != nil {
		panic(err)
	}
	m.Places = PlaceMap{}
	m.Transitions = TransitionMap{}
	m.Arcs = []Arc{}

	for label, p := range modelObject.Places {
		m.Places[label] = &Place{
			Label:    label,
			Offset:   int64(len(m.Places)),
			Position: Position{X: p.X, Y: p.Y},
			Initial:  p.Initial,
			Capacity: p.Capacity,
		}
	}

	for label, t := range modelObject.Transitions {
		var role = "default"
		if t.Role != "" {
			role = t.Role
		}
		m.Transitions[label] = &Transition{
			Label:    label,
			Position: Position{X: t.X, Y: t.Y},
			Role:     Role{Label: role},
			Delta:    m.EmptyVector(),
			Guards:   GuardMap{},
		}
	}

	for _, a := range modelObject.Arcs {
		source := m.Node(a.Source)
		target := m.Node(a.Target)
		if a.Inhibit {
			if source.IsPlace() {
				if !target.IsTransition() {
					panic(BadInhibitorTarget)
				}
				source.Guard(a.Weight, target)
			}
			if source.IsTransition() {
				if !target.IsPlace() {
					panic(BadInhibitorTarget)
				}
				source.Guard(a.Weight, target)
			}
		} else {
			source.Tx(a.Weight, target)
		}
	}

	m.Index()

	return true
}

func (m *Model) UnzipUrl(url string) (obj string, ok bool) {
	queryString := ""
	ok = false
	if i := strings.Index(url, "?"); i > -1 {
		queryString = url[i+1:]
		url = url[:i]
	}
	z := ""
	for _, param := range strings.Split(queryString, "&") {
		if strings.HasPrefix(param, "z=") {
			z = param[2:]
		}
	}
	// base64 decode z=
	if z != "" {
		decoded := make([]byte, len(z))
		_, err := b64.StdEncoding.Decode(decoded, []byte(z))
		if err != nil {
			panic(err)
		}
		// open zip archive
		zipReader, zipErr := zip.NewReader(bytes.NewReader(decoded), int64(len(decoded)))
		if zipErr != nil {
			panic(zipErr)
		}
		for _, file := range zipReader.File {
			if file.Name != "model.json" {
				continue
			}
			fileReader, err := file.Open()
			if err != nil {
				panic(err)
			}
			buf := new(bytes.Buffer)
			buf.ReadFrom(fileReader)
			obj = buf.String()
		}
	}
	ok = m.loadJsonDefinition(obj)
	return obj, ok
}

func (m *Model) Guard(source Node, target Node, weight int64) {
	if weight < 0 {
		panic(BadWeight)
	}
	if source.IsTransition() {
		if !target.IsPlace() {
			panic(BadInhibitorSource)
		}
		m.Arcs = append(m.Arcs, Arc{
			Source:    source,
			Target:    target,
			Weight:    weight,
			Inhibitor: true,
			Inverted:  true,
		})
	}
	if source.IsPlace() {
		if !target.IsTransition() {
			panic(BadInhibitorTarget)
		}
		m.Arcs = append(m.Arcs, Arc{
			Source:    source,
			Target:    target,
			Weight:    weight,
			Inhibitor: true,
			Inverted:  false,
		})
	}
}

func (m *Model) Node(oid string) Node {
	if m.Places[oid] != nil {
		return &node{
			m:     m,
			Place: m.Places[oid],
		}
	}
	if m.Transitions[oid] != nil {
		return &node{
			m:          m,
			Transition: m.Transitions[oid],
		}
	}
	return nil
}

// Graph repopulates Arcs using delta vectors and guards
func (m *Model) Graph() Editor {
	placeMap := make(map[int64]string)
	for label, p := range m.Places {
		placeMap[p.Offset] = label
	}
	m.Arcs = []Arc{}
	for _, t := range m.Transitions {
		for offset, d := range t.Delta {
			if d < 0 {
				m.Arcs = append(m.Arcs, Arc{
					Source: &node{
						m:     m,
						Place: m.Places[placeMap[int64(offset)]],
					},
					Target: &node{
						m:          m,
						Transition: t,
					},
					Weight: 0 - d,
				})
			} else if d > 0 {
				m.Arcs = append(m.Arcs, Arc{
					Target: &node{
						m:     m,
						Place: m.Places[placeMap[int64(offset)]],
					},
					Source: &node{
						m:          m,
						Transition: t,
					},
					Weight: d,
				})
			}
		}
		for _, g := range t.Guards {
			for offset, d := range g.Delta {
				if d < 0 {
					m.Arcs = append(m.Arcs, Arc{
						Source: &node{
							m:     m,
							Place: m.Places[placeMap[int64(offset)]],
						},
						Target: &node{
							m:          m,
							Transition: t,
						},
						Weight:    0 - d,
						Inhibitor: true,
					})

				} else if d != 0 {
					panic(BadInhibitorTarget)
				}
			}
		}
	}
	return m
}

// Index loads Arcs into delta vectors and guards
func (m *Model) Index() Editor {
	for _, t := range m.Transitions {
		t.Delta = m.EmptyVector()
	}
	for _, arc := range m.Arcs {
		if arc.Inhibitor {
			if arc.Inverted {
				g := &Guard{
					Label:    arc.Target.GetPlace().Label,
					Delta:    m.EmptyVector(),
					Inverted: true,
				}
				g.Delta[arc.Target.GetPlace().Offset] = 0 - arc.Weight
				arc.Source.GetTransition().Guards[g.Label] = g
			} else {
				g := &Guard{
					Label:    arc.Source.GetPlace().Label,
					Delta:    m.EmptyVector(),
					Inverted: false,
				}
				g.Delta[arc.Source.GetPlace().Offset] = 0 - arc.Weight
				arc.Target.GetTransition().Guards[g.Label] = g
			}
		} else {
			if arc.Source.IsPlace() {
				arc.Target.GetTransition().Delta[arc.Source.GetPlace().Offset] = 0 - arc.Weight
			} else {
				arc.Source.GetTransition().Delta[arc.Target.GetPlace().Offset] = arc.Weight
			}
		}
	}
	return m
}

func (m *Model) Net() *PetriNet {
	return m.PetriNet
}

func New(schema string) MetaModel {
	return &Model{
		PetriNet: &PetriNet{
			Schema:      schema,
			Places:      PlaceMap{},
			Transitions: TransitionMap{},
			Arcs:        []Arc{},
			Roles:       RoleMap{defaultRole.Label: defaultRole},
		},
	}
}

func (m *Model) Define(def ...func(declaration Declaration)) MetaModel {
	for _, defn := range def {
		defn(m)
	}
	m.Index()
	return m
}

// Execute run the m
func (m *Model) Execute(initialVec ...Vector) Process {

	sm := new(StateMachine)
	sm.m = m
	switch len(initialVec) {
	case 0:
		sm.state = m.InitialVector()
		sm.capacity = m.CapacityVector()
	case 1:
		sm.state = initialVec[0]
		sm.capacity = m.CapacityVector()
	case 2:
		sm.state = initialVec[0]
		sm.capacity = initialVec[1]
	default:
		panic(fmt.Sprintf(UnexpectedArguments, 2, len(initialVec)))
	}
	if len(sm.state) == 0 {
		sm.state = m.InitialVector()
	} else if len(sm.state) != len(sm.capacity) {
		sm.state = m.EmptyVector()
	}
	return sm
}

// Edit returns the internal interface used to edit and reindex a model
func (m *Model) Edit() Editor {
	return m
}

// Cell declares a new transition element
func (m *Model) Cell(def ...func(p *Place)) Node {
	p := &Place{
		Label:    m.PlaceSeq(),
		Offset:   int64(len(m.Places)),
		Position: Position{},
		Initial:  0,
		Capacity: 0,
	}
	for _, defn := range def {
		defn(p)
	}
	m.Places[p.Label] = p
	return &node{
		m:     m,
		Place: p,
	}
}

var defaultRole = Role{Label: "default"}

// Fn declares a new transition element
func (m *Model) Fn(def ...func(t *Transition)) Node {
	t := &Transition{
		Label:        m.TransitionSeq(),
		Position:     Position{},
		Role:         defaultRole,
		Delta:        Vector{},
		Guards:       GuardMap{},
		AllowReentry: false,
	}
	for _, defn := range def {
		defn(t)
	}
	m.Roles[t.Role.Label] = t.Role
	m.Transitions[t.Label] = t
	return &node{
		m:          m,
		Transition: t,
	}
}

// Arc connects places and transitions
// at runtime Arcs are indexed as adjacency matrix by converting Arcs to vectors
func (m *Model) Arc(source Node, target Node, weight int64) {
	if source.IsPlace() && target.IsPlace() {
		panic(BadArcPlace)
	}
	if source.IsTransition() && target.IsTransition() {
		panic(BadArcTransition)
	}
	if weight < 0 {
		panic(BadWeight)
	}
	m.Arcs = append(m.Arcs, Arc{
		Source:    source,
		Target:    target,
		Weight:    weight,
		Inhibitor: false,
	})
}

// PlaceSeq generates unique labels for places
func (m *Model) PlaceSeq() Label {
	i := 0
	for {
		label := fmt.Sprintf("place%v", i)
		if m.Places[label] == nil {
			return label
		} else {
			i++
		}
	}
}

// TransitionSeq generate unique labels for transitions
func (m *Model) TransitionSeq() Label {
	i := 0
	for {
		label := fmt.Sprintf("txn%v", i)
		if m.Transitions[label] == nil {
			return label
		} else {
			i++
		}
	}
}

type StateMachine struct {
	m        *Model
	state    Vector
	capacity Vector
}

func (sm *StateMachine) TestFire(op Op) (flag bool, msg string, out Vector) {
	txn := sm.m.Transitions[op.Action]
	if txn == nil {
		return false, UnknownAction, sm.GetState()
	}
	if op.Role != "" && txn.Role.Label != op.Role {
		return false, FailedRoleAssertion, sm.GetState()
	}
	if op.Multiple < 0 {
		return false, BadMultiple, sm.GetState()
	} else if op.Multiple == 0 {
		op.Multiple = defaultMultiple
	}
	isInhibited, label := sm.Inhibited(op)
	if isInhibited {
		return false, fmt.Sprintf(InhibitedTransition, label), out
	}
	flag, msg, out = Add(sm.state, txn.Delta, op.Multiple, sm.capacity)
	if !flag {
		return false, msg, out
	}
	return true, OK, out // REVIEW: match lua implementation to return Role
}

func (sm *StateMachine) Fire(op Op) (ok bool, msg string, out Vector) {
	ok, msg, out = sm.TestFire(op)
	if ok {
		for i, v := range out {
			sm.state[i] = v
		}
	}
	return ok, msg, out
}

func (sm *StateMachine) Inhibited(op Op) (inhibited bool, msg string) {
	tx := sm.m.Transitions[op.Action]
	if tx == nil {
		panic(UnknownAction)
	}
	for _, g := range tx.Guards {
		flag, _, _ := Add(sm.state, g.Delta, 1, sm.m.EmptyVector())
		if g.Inverted {
			if !flag {
				return true, g.Label
			}
		} else {
			if flag {
				return true, g.Label
			}
		}
	}
	return false, msg
}

func (sm *StateMachine) GetState() Vector {
	s := make([]int64, len(sm.state))
	copy(s, sm.state)
	return s
}

func (sm *StateMachine) TokenCount(label string) int64 {
	p := sm.m.Places[label]
	if p == nil {
		panic(ExpectedPlace)
	}
	return sm.state[p.Offset]
}
