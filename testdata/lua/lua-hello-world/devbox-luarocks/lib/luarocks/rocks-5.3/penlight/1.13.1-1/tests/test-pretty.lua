local pretty = require 'pl.pretty'
local utils = require 'pl.utils'
local test = require 'pl.test'
local asserteq, assertmatch = test.asserteq, test.assertmatch

t1 = {
    'one','two','three',{1,2,3},
    alpha=1,beta=2,gamma=3,['&']=true,[0]=false,
    _fred = {true,true},
    s = [[
hello dolly
you're so fine
]]
}

s = pretty.write(t1) --,' ',true)
t2,err = pretty.read(s)
if err then return print(err) end
asserteq(t1,t2)

res,err = pretty.read [[
  {
	['function'] = true,
	['do'] = true,
  }
]]
assert(res)

res,err = pretty.read [[
  {
    ['function'] = true,
    ['do'] = "no function here...",
  }
]]
assert(res)

res,err = pretty.read [[
  {
    ['function'] = true,
    ['do'] = function() return end
  }
]]
assertmatch(err,'cannot have functions in table definition')

res,err = pretty.load([[
-- comments are ok
a = 2
bonzo = 'dog'
t = {1,2,3}
]])

asserteq(res,{a=2,bonzo='dog',t={1,2,3}})

--- another potential problem is string functions called implicitly as methods--
res,err = pretty.read [[
{s = ('woo'):gsub('w','wwwwww'):gsub('w','wwwwww')}
]]

assertmatch(err,(_VERSION ~= "Lua 5.2") and 'attempt to index a string value' or "attempt to index constant 'woo'")

---- pretty.load has a _paranoid_ option
res,err = pretty.load([[
k = 0
for i = 1,1e12 do k = k + 1 end
]],{},true)

assertmatch(err,'looping not allowed')

-- Check to make sure that no spaces exist when write is told not to
local tbl = { "a", 2, "c", false, 23, 453, "poot", 34 }
asserteq( pretty.write( tbl, "" ), [[{"a",2,"c",false,23,453,"poot",34}]] )

-- Check that write correctly prevents cycles

local t1,t2 = {},{}
t1[1] = t1
asserteq( pretty.write(t1,""), [[{<cycle>}]] )
t1[1],t1[2],t2[1] = 42,t2,t1
asserteq( pretty.write(t1,""), [[{42,{<cycle>}}]] )

-- Check false positives in write's cycles prevention

t2 = {}
t1[1],t1[2] = t2,t2
asserteq( pretty.write(t1,""), [[{{},{}}]] )

-- Check that write correctly print table with non number or string as keys

t1 = { [true] = "boolean", [false] = "untrue", a = "a", b = "b", [1] = 1, [0] = 0 }
asserteq( pretty.write(t1,""), [[{1,["false"]="untrue",["true"]="boolean",a="a",b="b",[0]=0}]] )


-- Check number formatting
asserteq(pretty.write({1/0, -1/0, 0/0, 1, 1/2}, ""), "{Inf,-Inf,NaN,1,0.5}")

if _VERSION == "Lua 5.3" then
    asserteq(pretty.write({1.0}, ""), "{1.0}")
else
    asserteq(pretty.write({1.0}, ""), "{1}")
end

do  -- issue #203, item 3
  local t = {}; t[t] = 1
  pretty.write(t)  -- should not crash
end


-- pretty.write fails if an __index metatable raises an error #257
-- only applies to 5.3+ where iterators respect metamethods
do
  local t = setmetatable({},{
    __index = function(self, key)
      error("oops... couldn't find " .. tostring(key))
    end
  })
  asserteq(pretty.write(t), "{\n}")
end
