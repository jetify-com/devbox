-- Module containing classes
local class = require 'pl.class'
local utils = require 'pl.utils'
local error = error
if utils.lua51 then
    module 'animal'
else
    _ENV = {}
end

class.Animal()

function Animal:_init(name)
   self.name = name
end

function Animal:__tostring()
    return self.name..': '..self:speak()
end

class.Dog(Animal)

function Dog:speak()
    return 'bark'
end

class.Cat(Animal)

function Cat:_init(name,breed)
   self:super(name)  -- must init base!
   self.breed = breed
end

function Cat:speak()
    return 'meow'
end

-- you may declare the methods in-line like so;
-- note the meaning of `_base`!
class.Lion {
    _base = Cat;
    speak = function(self)
        return 'roar'
    end
}

-- a class may handle unknown methods with `catch`:
Lion:catch(function(self,name)
    return function() error("no such method "..name,2) end
end)

if not utils.lua51 then
   return _ENV
end
