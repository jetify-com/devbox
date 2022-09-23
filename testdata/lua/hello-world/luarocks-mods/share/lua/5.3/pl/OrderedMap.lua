--- OrderedMap, a map which preserves ordering.
--
-- Derived from `pl.Map`.
--
-- Dependencies: `pl.utils`, `pl.tablex`, `pl.class`, `pl.List`, `pl.Map`
-- @classmod pl.OrderedMap

local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local List = require 'pl.List'
local index_by,tsort,concat = tablex.index_by,table.sort,table.concat

local class = require 'pl.class'
local Map = require 'pl.Map'

local OrderedMap = class(Map)
OrderedMap._name = 'OrderedMap'

local rawset = rawset

--- construct an OrderedMap.
-- Will throw an error if the argument is bad.
-- @param t optional initialization table, same as for @{OrderedMap:update}
function OrderedMap:_init (t)
    rawset(self,'_keys',List())
    if t then
        local map,err = self:update(t)
        if not map then error(err,2) end
    end
end

local assert_arg,raise = utils.assert_arg,utils.raise

--- update an OrderedMap using a table.
-- If the table is itself an OrderedMap, then its entries will be appended.
-- if it s a table of the form `{{key1=val1},{key2=val2},...}` these will be appended.
--
-- Otherwise, it is assumed to be a map-like table, and order of extra entries is arbitrary.
-- @tab t a table.
-- @return the map, or nil in case of error
-- @return the error message
function OrderedMap:update (t)
    assert_arg(1,t,'table')
    if OrderedMap:class_of(t) then
       for k,v in t:iter() do
           self:set(k,v)
       end
    elseif #t > 0 then -- an array must contain {key=val} tables
       if type(t[1]) == 'table' then
           for _,pair in ipairs(t) do
               local key,value = next(pair)
               if not key then return raise 'empty pair initialization table' end
               self:set(key,value)
           end
       else
           return raise 'cannot use an array to initialize an OrderedMap'
       end
    else
       for k,v in pairs(t) do
           self:set(k,v)
       end
    end
   return self
end

--- set the key's value.   This key will be appended at the end of the map.
--
-- If the value is nil, then the key is removed.
-- @param key the key
-- @param val the value
-- @return the map
function OrderedMap:set (key,val)
    if rawget(self, key) == nil and val ~= nil then -- new key
        self._keys:append(key) -- we keep in order
        rawset(self,key,val)  -- don't want to provoke __newindex!
    else -- existing key-value pair
        if val == nil then
            self._keys:remove_value(key)
            rawset(self,key,nil)
        else
            self[key] = val
        end
    end
    return self
end

OrderedMap.__newindex = OrderedMap.set

--- insert a key/value pair before a given position.
-- Note: if the map already contains the key, then this effectively
-- moves the item to the new position by first removing at the old position.
-- Has no effect if the key does not exist and val is nil
-- @int pos a position starting at 1
-- @param key the key
-- @param val the value; if nil use the old value
function OrderedMap:insert (pos,key,val)
    local oldval = self[key]
    val = val or oldval
    if oldval then
        self._keys:remove_value(key)
    end
    if val then
        self._keys:insert(pos,key)
        rawset(self,key,val)
    end
    return self
end

--- return the keys in order.
-- (Not a copy!)
-- @return List
function OrderedMap:keys ()
    return self._keys
end

--- return the values in order.
-- this is relatively expensive.
-- @return List
function OrderedMap:values ()
    return List(index_by(self,self._keys))
end

--- sort the keys.
-- @func cmp a comparison function as for @{table.sort}
-- @return the map
function OrderedMap:sort (cmp)
    tsort(self._keys,cmp)
    return self
end

--- iterate over key-value pairs in order.
function OrderedMap:iter ()
    local i = 0
    local keys = self._keys
    local idx
    return function()
        i = i + 1
        if i > #keys then return nil end
        idx = keys[i]
        return idx,self[idx]
    end
end

--- iterate over an ordered map (5.2).
-- @within metamethods
-- @function OrderedMap:__pairs
OrderedMap.__pairs = OrderedMap.iter

--- string representation of an ordered map.
-- @within metamethods
function OrderedMap:__tostring ()
    local res = {}
    for i,v in ipairs(self._keys) do
        local val = self[v]
        local vs = tostring(val)
        if type(val) ~= 'number' then
            vs = '"'..vs..'"'
        end
        res[i] = tostring(v)..'='..vs
    end
    return '{'..concat(res,',')..'}'
end

return OrderedMap



