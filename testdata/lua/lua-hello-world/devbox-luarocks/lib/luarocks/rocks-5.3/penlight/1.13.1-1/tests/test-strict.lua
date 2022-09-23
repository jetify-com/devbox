require 'pl.compat' -- require this one before loading strict
local strict = require 'pl.strict'
local test = require 'pl.test'
local app = require 'pl.app'

-- in strict mode, you must assign to a global first, even if just nil.
test.assertraise(function()
   print(x)
   print 'ok?'
end,"variable 'x' is not declared")

-- can assign to globals in main (or from C extensions) but not anywhere else!
test.assertraise(function()
   Boo = 3
end,"assign to undeclared global 'Boo'")

Boo = true
Boo2 = nil

-- once declared, you can assign to globals from anywhere
(function() Boo = 42; Boo2 = 6*7 end)()

--- a module may use strict.module() to generate a simularly strict environment
-- (see lua/mymod.lua)
app.require_here 'lua'
local M = require 'mymod'

--- these are fine
M.answer()
M.question()

-- spelling mistakes become errors...
test.assertraise(function()
    print(M.Answer())
end,"variable 'Answer' is not declared in 'mymod'")

--- for the extra paranoid, you can choose to make all global tables strict...
strict.make_all_strict(_G)

test.assertraise(function()
    print(math.sine(1.2))
end,"variable 'sine' is not declared in 'math'")



-- module
do
  local testmodule = {
    hello = function() return "supremacy" end
  }
  -- make strict and allow extra field "world"
  strict.module("my_test", testmodule, { world = true })

  test.asserteq(testmodule.hello(), "supremacy")
  test.assertraise(function()
    print(testmodule.not_allowed_key)
  end, "variable 'not_allowed_key' is not declared in 'my_test'")

  test.asserteq(testmodule.world, nil)
  testmodule.world = "supremacy"
  test.asserteq(testmodule.world, "supremacy")


  -- table with a __newindex method
  local mod1 = strict.module("mod1", setmetatable(
    {
      hello = "world",
    }, {
      __newindex = function(self, key, value)
        if key == "Lua" then
          rawset(self, key, value)
        end
      end,
    }
  ))
  test.asserteq(mod1.hello, "world")
  mod1.Lua = "hello world"
  test.asserteq(mod1.Lua, "hello world")
  test.assertraise(function()
    print(mod1.not_allowed_key)
  end, "variable 'not_allowed_key' is not declared in 'mod1'")


  -- table with a __index method
  local mod1 = strict.module("mod1", setmetatable(
    {
      hello = "world",
    }, {
      __index = function(self, key)
        if key == "Lua" then
          return "rocks"
        end
      end,
    }
  ))
  test.asserteq(mod1.hello, "world")
  test.asserteq(mod1.Lua, "rocks")
  test.assertraise(function()
    print(mod1.not_allowed_key)
  end, "variable 'not_allowed_key' is not declared in 'mod1'")


  -- table with a __index table
  local mod1 = strict.module("mod1", setmetatable(
    {
      hello = "world",
    }, {
      __index = {
        Lua = "rocks!"
      }
    }
  ))
  test.asserteq(mod1.hello, "world")
  test.asserteq(mod1.Lua, "rocks!")
  test.assertraise(function()
    print(mod1.not_allowed_key)
  end, "variable 'not_allowed_key' is not declared in 'mod1'")

end


do
  -- closed_module
  -- what does this do? this does not seem a usefull function???

  local testmodule = {
    hello = function() return "supremacy" end
  }
  local M = strict.closed_module(testmodule, "my_test")

  -- read acces to original is granted, but not to the new one
  test.asserteq(testmodule.hello(), "supremacy")
  test.assertraise(function()
    print(M.hello())
  end, "variable 'hello' is not declared in 'my_test'")

  -- write access to both is granted
  testmodule.world = "domination"
  M.world = "domination"

  -- read acces to set field in original is granted, but not set
  test.asserteq(testmodule.world, nil)
  test.asserteq(M.world, "domination")

end
