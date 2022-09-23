-- testing Map functionality

local test = require 'pl.test'
local Map = require 'pl.Map'
local tablex = require 'pl.tablex'
local Set = require 'pl.Set'
local utils = require 'pl.utils'

local asserteq = test.asserteq
local cmp = tablex.compare_no_order



-- construction, plain
local m = Map{alpha=1,beta=2,gamma=3}

assert(cmp(
    m:values(),
    {1, 2, 3}
))

assert(cmp(
    m:keys(),
    {'alpha', 'beta', 'gamma'}
))

asserteq(
    m:items(),
    {
      {'alpha', 1},
      {'beta', 2},
      {'gamma', 3},
    }
)

asserteq (m:getvalues {'alpha','gamma'}, {1,3})



-- construction, from a set
local s = Set{'red','orange','green','blue'}
m = Map(s)

asserteq(
    m:items(),
    {
      {'blue', true},
      {'green', true},
      {'orange', true},
      {'red', true},
    }
)


-- iter()
m = Map{alpha=1,beta=2,gamma=3}
local t = {alpha=1,beta=2,gamma=3}
for k,v in m:iter() do
  asserteq(v, t[k])
  t[k] = nil
end
assert(next(t) == nil, "expected the table to be empty by now")



-- setdefault()
m = Map{alpha=1,beta=2,gamma=3}
local v = m:setdefault("charlie", 4)
asserteq(v, 4)
v = m:setdefault("alpha", 10)
asserteq(v, 1)
asserteq(
    m:items(),
    {
      {'alpha', 1},
      {'beta', 2},
      {'charlie', 4},
      {'gamma', 3},
    }
)
v = m:set("alpha", false)
v = m:setdefault("alpha", true)   -- falsy value should not be altered
asserteq(false, m:get("alpha"))



-- len()
m = Map{alpha=1,beta=2,gamma=3}
asserteq(3, m:len())
m = Map{}
asserteq(0, m:len())
m:set("charlie", 4)
asserteq(1, m:len())



-- set() & get()
m = Map{}
m:set("charlie", 4)
asserteq(4, m:get("charlie"))
m:set("charlie", 5)
asserteq(5, m:get("charlie"))
m:set("charlie", nil)
asserteq(nil, m:get("charlie"))



-- getvalues()
m = Map{alpha=1,beta=2,gamma=3}
local x = m:getvalues{"gamma", "beta"}
asserteq({3, 2}, x)



-- __eq()  -- equality
local m1 = Map{alpha=1,beta=2,gamma=3}
local m2 = Map{alpha=1,beta=2,gamma=3}
assert(m1 == m2)
m1 = Map()
m2 = Map()
assert(m1 == m2)



-- __tostring()
m = Map()
asserteq("{}", tostring(m))
m = Map{alpha=1}
asserteq("{alpha=1}", tostring(m))
m = Map{alpha=1,beta=2}
assert(({  -- test 2 versions, since we cannot rely on order
      ["{alpha=1,beta=2}"] = true,
      ["{beta=2,alpha=1}"] = true,
    })[tostring(m)])
