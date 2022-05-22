package lua

// Domain Specific Language for metamodels in Lua
const internalDsl = `
Models = {}

function domodel(schema, declaration)

	local def = {
		schema = schema,
		roles = {},
		places = {},
		transitions = {},
	}
	local arcs = {}

	local function fn (label, role, position)
		local transition = {
			label=label,
			role=role,
			position = position or { x=0, y=0, z=0 },
			guards={},
			delta={},
		}
		def.transitions[label] = transition
		return {
			transition = transition,
			tx = function(weight, target)
				assert(target, 'target is nil')
				assert(target.place, 'target node must be a place')
				table.insert(arcs, {
					source = { transition = transition },
					target = target, weight = weight
				})
			end,
		}
	end

	local place_count = 0 -- NOTE: lua starts w/ 1 index by default, this matches golang

	local function cell (label, initial, capacity, position)
		place_count = place_count + 1
		local place = {
			label=label,
			initial=initial or 0,
			capacity=capacity or 0,
			position=position or {},
			offset= place_count
		}
		def.places[label] = place

		local function tx(weight, target)
			table.insert(arcs, {
				source = { place = place },
				target = target,
				weight = weight or 1
			})
			assert(target.transition, 'target node must be a transition')
			return
		end

		local function guard (weight, target)
			table.insert(arcs, {
				source = { place = place },
				target = target,
				weight = weight,
				inhibit = true
			})
			assert(target.transition, 'target node must be a transition')
		end

		return {
			place = place,
			tx = tx,
			guard = guard,
		}
	end

	local function role (label)
		if not def.roles[label] then
			def.roles[label] = { label=label }
		end
		return def.roles[label]
	end

	declaration(fn, cell, role)

	local function empty_vector()
		local v = {}
		for _, p in pairs( def.places ) do
			v[p.offset] = 0
		end
		return v
	end

	for _, t in pairs( def.transitions ) do
		t.delta = empty_vector() -- right size all deltas
	end

	for _, arc in pairs( arcs ) do
		if (arc.inhibit) then
			local g = {
				label = arc.source.place.label,
				delta = empty_vector(),
			}
			g.delta[arc.source.place.offset] = 0-arc.weight
			arc.target.transition.guards[arc.source.place.label] = g
		else
			if (arc.source.transition) then
				arc.source.transition.delta[arc.target.place.offset] = arc.weight
			else
				arc.target.transition.delta[arc.source.place.offset] = 0-arc.weight
			end
		end
	end
	Models[schema] = def
end
`
