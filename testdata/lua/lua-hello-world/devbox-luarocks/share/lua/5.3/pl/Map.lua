--- A Map class.
--
--    > Map = require 'pl.Map'
--    > m = Map{one=1,two=2}
--    > m:update {three=3,four=4,two=20}
--    > = m == M{one=1,two=20,three=3,four=4}
--    true
--
-- Dependencies: `pl.utils`, `pl.class`, `pl.tablex`, `pl.pretty`
-- @classmod pl.Map

local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local stdmt = utils.stdmt
local deepcompare = tablex.deepcompare

local pretty_write = require 'pl.pretty' . write
local Map = stdmt.Map
local Set = stdmt.Set

local class = require 'pl.class'

-- the Map class ---------------------
class(nil,nil,Map)

function Map:_init (t)
    local mt = getmetatable(t)
    if mt == Set or mt == Map then
        self:update(t)
    else
        return t -- otherwise assumed to be a map-like table
    end
end


local function makelist(t)
    return setmetatable(t, require('pl.List'))
end

--- list of keys.
Map.keys = tablex.keys

--- list of values.
Map.values = tablex.values

--- return an iterator over all key-value pairs.
function Map:iter ()
    return pairs(self)
end

--- return a List of all key-value pairs, sorted by the keys.
function Map:items()
    local ls = makelist(tablex.pairmap (function (k,v) return makelist {k,v} end, self))
    ls:sort(function(t1,t2) return t1[1] < t2[1] end)
    return ls
end

--- set a value in the map if it doesn't exist yet.
-- @param key the key
-- @param default value to set
-- @return the value stored in the map (existing value, or the new value)
function Map:setdefault(key, default)
    local val = self[key]
    if val ~= nil then
        return val
    end
    self:set(key,default)
   return default
end

--- size of map.
-- note: this is a relatively expensive operation!
-- @class function
-- @name Map:len
Map.len = tablex.size

--- put a value into the map.
-- This will remove the key if the value is `nil`
-- @param key the key
-- @param val the value
function Map:set (key,val)
    self[key] = val
end

--- get a value from the map.
-- @param key the key
-- @return the value, or nil if not found.
function Map:get (key)
    return rawget(self,key)
end

local index_by = tablex.index_by

--- get a list of values indexed by a list of keys.
-- @param keys a list-like table of keys
-- @return a new list
function Map:getvalues (keys)
    return makelist(index_by(self,keys))
end

--- update the map using key/value pairs from another table.
-- @tab table
-- @function Map:update
Map.update = tablex.update

--- equality between maps.
-- @within metamethods
-- @tparam Map m another map.
function Map:__eq (m)
    -- note we explicitly ask deepcompare _not_ to use __eq!
    return deepcompare(self,m,true)
end

--- string representation of a map.
-- @within metamethods
function Map:__tostring ()
    return pretty_write(self,'')
end

return Map
