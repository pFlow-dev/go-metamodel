package js

var internalDsl = `
Model = ""
function domodel(schema, declaration) {
	def = {
		schema: schema,
		roles: {},
		places: {},
		transitions: {},
		arcs: [],
	};

	function assert(flag, msg) {
		if (! flag) {
			throw new Error(msg);
		}
	}

	function fn(label, role, position) {
		transition =  { label: label, role: role, position: position, guards: {}, delta: [] };
		def.transitions[label] = transition;

		function tx(weight, target) {
			assert(target, "target is null" );
			assert(target.place, "target node must be a place");
			def.arcs.push({
				source: { transition: label },
				target: { place: target.place.label },
				weight: weight,
				inhibit: false
			});
		}
		return {
			transition: transition,
			tx: tx
		};
	}
	placeCount = 0;
	function cell(label, initial, capacity, position) {
		place = {
			label: label,
			initial: initial || 0,
			capacity: capacity || 0,
			position: position || {},
			offset: placeCount
		};
		placeCount = placeCount + 1; // NOTE: js arrays begin with index 0
		def.places[label] = place;

		function tx(weight, target) {
			def.arcs.push({
				source: { place: label },
				target: { transition: target.transition.label },
				weight: weight || 1,
				inhibit: false
			});
			assert(target.transition, "target node must be a transition");
		}

		function guard(weight, target) {
			def.arcs.push({
				source: { place: place.label },
				target: { transition: target.transition.label },
				weight: weight,
				inhibit: true
			});
			assert(target.transition, "target node must be a transition");
		}
		return { place: place, tx: tx, guard: guard };
	}

	function role(label) {
		if (!def.roles[label]) {
			def.roles[label] = { label: label };
		}
		return def.roles[label];
	}
	function emptyVector() {
		v = [];
		for (i in def.places) {
			p = def.places[i];
			v[p.offset] = 0;
		}
		return v;
	}
	function initialVector() {
		v = [];
		for (i in def.places) {
			p = def.places[i];
			v[p.offset] = p.initial;
		}
		return v;
	}
	function capacityVector() {
		v = [];
		for (i in def.places) {
			p = def.places[i];
			v[p.offset] = p.capacity;
		}
		return v;
	}
	function index() {
		for (i in def.transitions) {
			def.transitions[i].delta = emptyVector(); // right size all deltas
		}
		ok = true;
		for (i in def.arcs) {
			arc = def.arcs[i];
			if (arc.inhibit) {
				g = {
					label: arc.target.transition,
					delta: emptyVector(),
				};
				g.delta[def.places[arc.source.place].offset] = 0 - arc.weight;
				def.transitions[arc.target.transition].guards[arc.source.place] = g;
			} else if (arc.source.place) {
				def.transitions[arc.target.transition].delta[def.places[arc.source.place].offset] = 0 - arc.weight;
			} else if (arc.source.transition) {
				def.transitions[arc.source.transition].delta[def.places[arc.target.place].offset] = arc.weight;
			} else {
				ok = false;
			}
		}
		return ok;
	}

	function vectorAdd(state, delta, multiple) {
		cap = capacityVector();
		out = [];
		ok = true;
		for (i in state) {
			out[i] = state[i] + delta[i] * multiple;
			if (out[i] < 0) {
				ok = false; // underflow: contains negative
			} else if (cap[i] > 0 && cap[i] - out[i] < 0 ) {
				ok = false; // overflow: exceeds capacity
			}
		}
		return { out: out, ok: ok };
	}

	function guardFails(state, action, multiple) {
		assert(action, "action is nil");
		t = def.transitions[action];
		assert(t, "action not found: " + action );
		for (i in t.guards) {
			guard = t.guards[i];
			res = vectorAdd(state, guard.delta, multiple);
			if (res.ok) {
				return true; // inhibitor active
			}
		}
		return false; // inhibitor inactive
	}

	function testFire(state, action, multiple) {
		t = def.transitions[action];
		if (guardFails(state, action, multiple) ) {
			return { out: null, ok: false, role: t.role.label };
		}
		res = vectorAdd(state, t.delta, multiple);
		return { out: res.out, ok: res.ok, role: t.role.label };
	}

	function fire(state, action, multiple, resolve, reject) {
		res = testFire(state, action, multiple);
		console.log({testFire: res});
		if (res.ok) {
			for (i in res.out) {
				state[i] = res.out[i];
			}
			if (resolve) {
				resolve(res);
			}
		} else if (reject) {
			reject(res);
		}
		return res;
	}

	if (declaration) {
		declaration(fn, cell, role);
		if (!index()) {
			throw new Error("invalid declaration");
		}
	}

	Model = JSON.stringify(def)

	return {
		dsl: { fn: fn, cell: cell, role: role },
		def: def,
		index: index,
		guardFails: guardFails,
		emptyVector: emptyVector,
		initialVector: initialVector,
		capacityVector: capacityVector,
		testFire: testFire,
		fire: fire,
	};
}
`
