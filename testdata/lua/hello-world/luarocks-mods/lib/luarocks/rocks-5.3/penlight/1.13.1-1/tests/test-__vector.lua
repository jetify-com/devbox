---- deriving specialized classes from List
-- illustrating covariance of List methods
local test = require 'pl.test'
local class = require 'pl.class'
local types = require 'pl.types'
local operator = require 'pl.operator'
local List = require 'pl.List'

local asserteq = test.asserteq

class.Vector(List)


function Vector.range (x1,x2,delta)
   return Vector(List.range(x1,x2,delta))
end

local function vbinop (op,v1,v2,scalar)
   if not Vector:class_of(v1) then
      v2, v1 = v1, v2
   end
   if type(v2) ~= 'table' then
      return v1:map(op,v2)
   else
      if scalar then error("operation not permitted on two vectors",3) end
      if #v1 ~= #v2 then error("vectors have different lengths",3) end
      return v1:map2(op,v2)
   end
end

function Vector.__add (v1,v2)
   return vbinop(operator.add,v1,v2)
end

function Vector.__sub (v1,v2)
   return vbinop(operator.sub,v1,v2)
end

function Vector.__mul (v1,v2)
   return vbinop(operator.mul,v1,v2,true)
end

function Vector.__div (v1,v2)
   return vbinop(operator.div,v1,v2,true)
end

function Vector.__unm (v)
   return v:map(operator.unm)
end

Vector:catch(List.default_map_with(math))

v = Vector()

assert(v:is_a(Vector))
assert(Vector:class_of(v))

v:append(10)
v:append(20)
asserteq(1+v,v+1)

-- covariance: the inherited Vector.map returns a Vector
asserteq(List{1,2} + v:map '2*_',{21,42})

u = Vector{1,2}

asserteq(v + u,{11,22})
asserteq(v - u,{9,18})
asserteq (v - 1, {9,19})
asserteq(2 * v, {20,40})
-- print(v * v) -- throws error: not permitted
-- print(v + Vector{1,2,3}) -- throws error: different lengths
asserteq(2*v + u, {21,42})
asserteq(-v, {-10,-20})

-- Vector.slice returns the Right Thing due to covariance
asserteq(
   Vector.range(0,1,0.1):slice(1,3)+1,
   {1,1.1,1.2},
   1e-8)

u:transform '_+1'
asserteq(u,{2,3})

u = Vector.range(0,1,0.1)
asserteq(
   u:map(math.sin),
   {0,0.0998,0.1986,0.2955,0.3894,0.4794,0.5646,0.6442,0.7173,0.7833,0.8414},
0.001)

-- unknown Vector methods are assumed to be math.* functions
asserteq(Vector{-1,2,3,-4}:abs(),Vector.range(1,4))

local R = Vector.range

-- concatenating two Vectors returns another vector (covariance again)
-- note the operator precedence here...
asserteq((
   R(0,1,0.1)..R(1.2,2,0.2)) + 1,
   {1,1.1,1.2,1.3,1.4,1.5,1.6,1.7,1.8,1.9,2,2.2,2.4,2.6,2.8,3},
   1e-8)


class.Strings(List)

Strings:catch(List.default_map_with(string))

ls = Strings{'one','two','three'}
asserteq(ls:upper(),{'ONE','TWO','THREE'})
asserteq(ls:sub(1,2),{'on','tw','th'})

-- all map operations on specialized lists
-- results in another list of that type! This isn't necessarily
-- what you want.
local sizes = ls:map '#'
asserteq(sizes, {3,3,5})
asserteq(types.type(sizes),'Strings')
asserteq(sizes:is_a(Strings),true)
sizes = Vector:cast(sizes)
asserteq(types.type(sizes),'Vector')
asserteq(sizes+1,{4,4,6})


