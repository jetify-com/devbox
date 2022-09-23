local class = require 'pl.class'
local A = class()
function A:_init()
  self.init_chain = "A"
end
local B = class(A)
local C = class(B)
function C:_init()
  self:super()
  self.init_chain = self.init_chain.."C"
end
local D = class(C)
local E = class(D)
function E:_init()
  self:super()
  self.init_chain = self.init_chain.."E"
end
local F = class(E)
local G = class(F)
function G:_init()
  self:super()
  self.init_chain = self.init_chain.."G"
end

local i = G()
assert(i.init_chain == "ACEG")
