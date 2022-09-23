local List = require 'pl.List'
local class = require 'pl.class'
local test = require 'pl.test'
local asserteq, T = test.asserteq, test.tuple

-- note that a _plain table_ is made directly into a list
local t = {10,20,30}
local ls = List(t)
asserteq(t,ls)

asserteq(List({}):reverse(), {})
asserteq(List({1}):reverse(), {1})
asserteq(List({1,2}):reverse(), {2,1})
asserteq(List({1,2,3}):reverse(), {3,2,1})
asserteq(List({1,2,3,4}):reverse(), {4,3,2,1})

-- you may derive classes from pl.List, and the result is covariant.
-- That is, slice() etc will return a list of the derived type, not List.

local NA = class(List)

local function mapm(a1,op,a2)
  local M = type(a2)=='table' and List.map2 or List.map
  return M(a1,op,a2)
end

--- elementwise arithmetric operations
function NA.__unm(a) return a:map '|X|-X' end
function NA.__pow(a,s) return a:map '|X,Y|X^Y' end
function NA.__add(a1,a2) return mapm(a1,'|X,Y|X+Y',a2) end
function NA.__sub(a1,a2) return mapm(a1,'|X,Y|X-Y',a2) end
function NA.__div(a1,a2) return mapm(a1,'|X,Y|X/Y',a2) end
function NA.__mul(a1,a2) return mapm(a2,'|X,Y|X*Y',a1) end

function NA:minmax ()
    local min,max = math.huge,-math.huge
    for i = 1,#self do
        local val = self[i]
        if val > max then max = val end
        if val < min then min = val end
    end
    return min,max
end

function NA:sum ()
    local res = 0
    for i = 1,#self do
        res = res + self[i]
    end
    return res
end

function NA:normalize ()
    return self:transform('|X,Y|X/Y',self:sum())
end

n1 = NA{10,20,30}
n2 = NA{1,2,3}
ns = n1 + 2*n2

asserteq(List:class_of(ns),true)
asserteq(NA:class_of(ns),true)
asserteq(ns:is_a(NA),true)
asserteq(ns,{12,24,36})
min,max = ns:slice(1,2):minmax()
asserteq(T(min,max),T(12,24))

asserteq(n1:normalize():sum(),1,1e-8)
