--- A Set class.
--
--     > Set = require 'pl.Set'
--     > = Set{'one','two'} == Set{'two','one'}
--     true
--     > fruit = Set{'apple','banana','orange'}
--     > = fruit['banana']
--     true
--     > = fruit['hazelnut']
--     nil
--     > colours = Set{'red','orange','green','blue'}
--     > = fruit,colours
--     [apple,orange,banana]   [blue,green,orange,red]
--     > = fruit+colours
--     [blue,green,apple,red,orange,banana]
--     [orange]
--     > more_fruits = fruit + 'apricot'
--     > = fruit*colours
--    > =  more_fruits, fruit
--    [banana,apricot,apple,orange]	[banana,apple,orange]
--
-- Dependencies: `pl.utils`, `pl.tablex`, `pl.class`, `pl.Map`, (`pl.List` if __tostring is used)
-- @classmod pl.Set

local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local array_tostring, concat = utils.array_tostring, table.concat
local merge,difference = tablex.merge,tablex.difference
local Map = require 'pl.Map'
local class = require 'pl.class'
local stdmt = utils.stdmt
local Set = stdmt.Set

-- the Set class --------------------
class(Map,nil,Set)

-- note: Set has _no_ methods!
Set.__index = nil

local function makeset (t)
    return setmetatable(t,Set)
end

--- create a set. <br>
-- @param t may be a Set, Map or list-like table.
-- @class function
-- @name Set
function Set:_init (t)
    t = t or {}
    local mt = getmetatable(t)
    if mt == Set or mt == Map then
        for k in pairs(t) do self[k] = true end
    else
        for _,v in ipairs(t) do self[v] = true end
    end
end

--- string representation of a set.
-- @within metamethods
function Set:__tostring ()
    return '['..concat(array_tostring(Set.values(self)),',')..']'
end

--- get a list of the values in a set.
-- @param self a Set
-- @function Set.values
-- @return a list
Set.values = Map.keys

--- map a function over the values of a set.
-- @param self a Set
-- @param fn a function
-- @param ... extra arguments to pass to the function.
-- @return a new set
function Set.map (self,fn,...)
    fn = utils.function_arg(1,fn)
    local res = {}
    for k in pairs(self) do
        res[fn(k,...)] = true
    end
    return makeset(res)
end

--- union of two sets (also +).
-- @param self a Set
-- @param set another set
-- @return a new set
function Set.union (self,set)
    return merge(self,set,true)
end

--- modifies '+' operator to allow addition of non-Set elements
--- Preserves +/- semantics - does not modify first argument.
local function setadd(self,other)
    local mt = getmetatable(other)
    if mt == Set or mt == Map then
        return Set.union(self,other)
    else
        local new = Set(self)
        new[other] = true
        return new
    end
end

--- union of sets.
-- @within metamethods
-- @function Set.__add

Set.__add = setadd

--- intersection of two sets (also *).
-- @param self a Set
-- @param set another set
-- @return a new set
-- @usage
-- > s = Set{10,20,30}
-- > t = Set{20,30,40}
-- > = t
-- [20,30,40]
-- > = Set.intersection(s,t)
-- [30,20]
-- > = s*t
-- [30,20]

function Set.intersection (self,set)
    return merge(self,set,false)
end

--- intersection of sets.
-- @within metamethods
-- @function Set.__mul
Set.__mul = Set.intersection

--- new set with elements in the set that are not in the other (also -).
-- @param self a Set
-- @param set another set
-- @return a new set
function Set.difference (self,set)
    return difference(self,set,false)
end

--- modifies "-" operator to remove non-Set values from set.
--- Preserves +/- semantics - does not modify first argument.
local function setminus (self,other)
    local mt = getmetatable(other)
    if mt == Set or mt == Map then
        return Set.difference(self,other)
    else
        local new = Set(self)
        new[other] = nil
        return new
    end
end

--- difference of sets.
-- @within metamethods
-- @function Set.__sub
Set.__sub = setminus

-- a new set with elements in _either_ the set _or_ other but not both (also ^).
-- @param self a Set
-- @param set another set
-- @return a new set
function Set.symmetric_difference (self,set)
    return difference(self,set,true)
end

--- symmetric difference of sets.
-- @within metamethods
-- @function Set.__pow
Set.__pow = Set.symmetric_difference

--- is the first set a subset of the second (also <)?.
-- @param self a Set
-- @param set another set
-- @return true or false
function Set.issubset (self,set)
    for k in pairs(self) do
        if not set[k] then return false end
    end
    return true
end

--- first set subset of second?
-- @within metamethods
-- @function Set.__lt
Set.__lt = Set.issubset

--- is the set empty?.
-- @param self a Set
-- @return true or false
function Set.isempty (self)
    return next(self) == nil
end

--- are the sets disjoint? (no elements in common).
-- Uses naive definition, i.e. that intersection is empty
-- @param s1 a Set
-- @param s2 another set
-- @return true or false
function Set.isdisjoint (s1,s2)
    return Set.isempty(Set.intersection(s1,s2))
end

--- size of this set (also # for 5.2).
-- @param s a Set
-- @return size
-- @function Set.len
Set.len = tablex.size

--- cardinality of set (5.2).
-- @within metamethods
-- @function Set.__len
Set.__len = Set.len

--- equality between sets.
-- @within metamethods
function Set.__eq (s1,s2)
    return Set.issubset(s1,s2) and Set.issubset(s2,s1)
end

return Set
