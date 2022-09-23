-- animal.lua
require 'pl.app'.require_here 'lua'

local test = require 'pl.test'
local asserteq = test.asserteq

local A = require 'animal'

local fido, felix, leo
fido = A.Dog('Fido')
felix = A.Cat('Felix','Tabby')
leo = A.Lion('Leo','African')

asserteq(fido:speak(),'bark')
asserteq(felix:speak(),'meow')
asserteq(leo:speak(),'roar')

asserteq(tostring(leo),'Leo: roar')

test.assertraise(function() leo:circus_act() end, "no such method circus_act")

asserteq(leo:is_a(A.Animal),true)
asserteq(leo:is_a(A.Dog),false)
asserteq(leo:is_a(A.Cat),true)

asserteq(A.Dog:class_of(leo),false)
asserteq(A.Cat:class_of(leo),true)
asserteq(A.Lion:class_of(leo),true)
