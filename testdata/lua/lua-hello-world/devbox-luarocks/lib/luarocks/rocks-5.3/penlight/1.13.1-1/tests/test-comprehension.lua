-- test-comprehension.lua
-- test of comprehension.lua
local utils = require 'pl.utils'
local comp = require 'pl.comprehension' . new()
local asserteq = require 'pl.test' . asserteq

-- test of list building
asserteq(comp 'x for x' {}, {})
asserteq(comp 'x for x' {2,3}, {2,3})
asserteq(comp 'x^2 for x' {2,3}, {2^2,3^2})
asserteq(comp 'x for x if x % 2 == 0' {4,5,6,7}, {4,6})
asserteq(comp '{x,y} for x for y if x>2 if y>4' ({2,3},{4,5}), {{3,5}})

-- test of table building
local t = comp 'table(x,x+1 for x)' {3,4}
assert(t[3] == 3+1 and t[4] == 4+1)
local t = comp 'table(x,x+y for x for y)' ({3,4}, {2})
assert(t[3] == 3+2 and t[4] == 4+2)
local t = comp 'table(v,k for k,v in pairs(_1))' {[3]=5, [5]=7}
assert(t[5] == 3 and t[7] == 5)

-- test of sum
assert(comp 'sum(x for x)' {} == 0)
assert(comp 'sum(x for x)' {2,3} == 2+3)
assert(comp 'sum(x^2 for x)' {2,3} == 2^2+3^2)
assert(comp 'sum(x*y for x for y)' ({2,3}, {4,5}) == 2*4+2*5+3*4+3*5)
assert(comp 'sum(x^2 for x if x % 2 == 0)' {4,5,6,7} == 4^2+6^2)
assert(comp 'sum(x*y for x for y if x>2 if y>4)' ({2,3}, {4,5}) == 3*5)

-- test of min/max
assert(comp 'min(x for x)' {3,5,2,4} == 2)
assert(comp 'max(x for x)' {3,5,2,4} == 5)

-- test of placeholder parameters --
assert(comp 'sum(x^_1 + _3 for x if x >= _4)' (2, nil, 3, 4, {3,4,5})
       == 4^2+3 + 5^2+3)

-- test of for =
assert(comp 'sum(x^2 for x=2,3)' () == 2^2+3^2)
assert(comp 'sum(x^2 for x=2,6,1+1)' () == 2^2+4^2+6^2)
assert(comp 'sum(x*y*z for x=1,2 for y=3,3 for z)' {5,6} ==
  1*3*5 + 2*3*5 + 1*3*6 + 2*3*6)
assert(comp 'sum(x*y*z for z for x=1,2 for y=3,3)' {5,6} ==
  1*3*5 + 2*3*5 + 1*3*6 + 2*3*6)

-- test of for in
assert(comp 'sum(i*v for i,v in ipairs(_1))' {2,3} == 1*2+2*3)
assert(comp 'sum(i*v for i,v in _1,_2,_3)' (ipairs{2,3}) == 1*2+2*3)

-- test of difficult syntax
asserteq(comp '" x for x " for x' {2}, {' x for x '})
asserteq(comp 'x --[=[for x\n\n]=] for x' {2}, {2})
asserteq(comp '(function() for i = 1,1 do return x*2 end end)() for x'
               {2}, {4})
assert(comp 'sum(("_5" and x)^_1 --[[_6]] for x)' (2, {4,5}) == 4^2 + 5^2)

-- error checking
assert(({pcall(function() comp 'x for __result' end)})[2]
       :find'not contain __ prefix')

-- environment.
-- Note: generated functions are set to the environment of the 'new' call.
 asserteq(5,(function()
      local env = {d = 5}
      local comp = comp.new(env)
      return comp 'sum(d for x)' {1}
 end)());
print 'DONE'
