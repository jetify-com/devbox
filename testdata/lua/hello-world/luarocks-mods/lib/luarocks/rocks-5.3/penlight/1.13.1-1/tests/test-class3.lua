-- another way to define classes. Works particularly well
-- with Moonscript
local class = require('pl.class')
local A = class{
  _init = function(self, name)
    self.name = name
  end,
  greet = function(self)
    return "hello " .. self.name
  end,
  __tostring = function(self)
    return self.name
  end
}

local B = class{
  _base = A,
  
  greet = function(self)
    return "hola " .. self.name
  end
}

local a = A('john')
assert(a:greet()=="hello john")
assert(tostring(a) == "john")
local b = B('juan')
assert(b:greet()=="hola juan")
assert(tostring(b)=="juan")
