--- MultiMap, a Map which has multiple values per key.
--
-- Dependencies: `pl.utils`, `pl.class`, `pl.List`, `pl.Map`
-- @classmod pl.MultiMap

local utils = require 'pl.utils'
local class = require 'pl.class'
local List = require 'pl.List'
local Map = require 'pl.Map'

-- MultiMap is a standard MT
local MultiMap = utils.stdmt.MultiMap

class(Map,nil,MultiMap)
MultiMap._name = 'MultiMap'

function MultiMap:_init (t)
    if not t then return end
    self:update(t)
end

--- update a MultiMap using a table.
-- @param t either a Multimap or a map-like table.
-- @return the map
function MultiMap:update (t)
    utils.assert_arg(1,t,'table')
    if Map:class_of(t) then
        for k,v in pairs(t) do
            self[k] = List()
            self[k]:append(v)
        end
    else
        for k,v in pairs(t) do
            self[k] = List(v)
        end
    end
end

--- add a new value to a key.  Setting a nil value removes the key.
-- @param key the key
-- @param val the value
-- @return the map
function MultiMap:set (key,val)
    if val == nil then
        self[key] = nil
    else
        if not self[key] then
            self[key] = List()
        end
        self[key]:append(val)
    end
end

return MultiMap
