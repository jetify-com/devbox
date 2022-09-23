local tablex = require 'pl.tablex'
local utils = require ('pl.utils')
local L = utils.string_lambda
local test = require('pl.test')
-- bring tablex funtions into global namespace
utils.import(tablex)
local asserteq = test.asserteq

local cmp = deepcompare

function asserteq_no_order (x,y)
    if not compare_no_order(x,y) then
        test.complain(x,y,"these lists contained different elements")
    end
end


asserteq(
    copy {10,20,30},
    {10,20,30}
)

asserteq(
    deepcopy {10,20,{30,40}},
    {10,20,{30,40}}
)

local t = {
    a = "hello",
    b = {
        c = "world"
    }
}
t.b.d = t.b

local tcopy = {
    a = "hello",
    b = {
        c = "world"
    }
}
tcopy.b.d = tcopy.b

asserteq(
	deepcopy(t),
	tcopy
)

asserteq(
    pairmap(function(i,v) return v end,{10,20,30}),
    {10,20,30}
)

asserteq_no_order(
    pairmap(L'_',{fred=10,bonzo=20}),
    {'fred','bonzo'}
)

asserteq_no_order(
    pairmap(function(k,v) return v end,{fred=10,bonzo=20}),
    {10,20}
)

asserteq_no_order(
    pairmap(function(i,v) return v,i end,{10,20,30}),
    {10,20,30}
)

asserteq(
    pairmap(function(k,v) return {k,v},k end,{one=1,two=2}),
    {one={'one',1},two={'two',2}}
)
-- same as above, using string lambdas
asserteq(
    pairmap(L'|k,v|{k,v},k',{one=1,two=2}),
    {one={'one',1},two={'two',2}}
)


asserteq(
    map(function(v) return v*v end,{10,20,30}),
    {100,400,900}
)

-- extra arguments to map() are passed to the function; can use
-- the abbreviations provided by pl.operator
asserteq(
    map('+',{10,20,30},1),
    {11,21,31}
)

asserteq(
    map(L'_+1',{10,20,30}),
    {11,21,31}
)

-- map2 generalizes for operations on two tables
asserteq(
    map2(math.max,{1,2,3},{0,4,2}),
    {1,4,3}
)

-- mapn operates over an arbitrary number of input tables (but use map2 for n=2)
asserteq(
    mapn(function(x,y,z) return x+y+z end, {1,2,3},{10,20,30},{100,200,300}),
    {111,222,333}
)

asserteq(
    mapn(math.max, {1,20,300},{10,2,3},{100,200,100}),
    {100,200,300}
)

asserteq(
    count_map({"foo", "bar", "foo", "baz"}),
    {foo = 2, bar = 1, baz = 1}
)

asserteq(
    zip({10,20,30},{100,200,300}),
    {{10,100},{20,200},{30,300}}
)

assert(compare_no_order({1,2,3,4},{2,1,4,3})==true)
assert(compare_no_order({1,2,3,4},{2,1,4,4})==false)

asserteq(range(10,9),{})
asserteq(range(10,10),{10})
asserteq(range(10,11),{10,11})

-- update inserts key-value pairs from the second table
t1 = {one=1,two=2}
t2 = {three=3,two=20,four=4}
asserteq(update(t1,t2),{one=1,three=3,two=20,four=4})

-- the difference between move and icopy is that the second removes
-- any extra elements in the destination after end of copy
-- 3rd arg is the index to start in the destination, defaults to 1
asserteq(move({1,2,3,4,5,6},{20,30}),{20,30,3,4,5,6})
asserteq(move({1,2,3,4,5,6},{20,30},2),{1,20,30,4,5,6})
asserteq(icopy({1,2,3,4,5,6},{20,30},2),{1,20,30})
-- 5th arg determines how many elements to copy (default size of source)
asserteq(icopy({1,2,3,4,5,6},{20,30},2,1,1),{1,20})
-- 4th arg is where to stop copying from the source (default s to 1)
asserteq(icopy({1,2,3,4,5,6},{20,30},2,2,1),{1,30})

-- whereas insertvalues works like table.insert, but inserts a range of values
-- from the given table.
asserteq(insertvalues({1,2,3,4,5,6},2,{20,30}),{1,20,30,2,3,4,5,6})
asserteq(insertvalues({1,2},{3,4}),{1,2,3,4})

-- the 4th arg of move and icopy gives the start index in the source table
asserteq(move({1,2,3,4,5,6},{20,30},2,2),{1,30,3,4,5,6})
asserteq(icopy({1,2,3,4,5,6},{20,30},2,2),{1,30})

t = {1,2,3,4,5,6}
move(t,{20,30},2,1,1)
asserteq(t,{1,20,3,4,5,6})
set(t,0,2,3)
asserteq(t,{1,0,0,4,5,6})
insertvalues(t,1,{10,20})
asserteq(t,{10,20,1,0,0,4,5,6})

asserteq(merge({10,20,30},{nil, nil, 30, 40}), {[3]=30})
asserteq(merge({10,20,30},{nil, nil, 30, 40}, true), {10,20,30,40})


-- Function to check that the order of elements returned by the iterator
-- match the order of the elements in the list.
function assert_iter_order(iter,l)
   local i = 0
   for k,v in iter do
      i = i + 1
      asserteq(k,l[i][1])
      asserteq(v,l[i][2])
   end
end

local t = {a=10,b=9,c=8,d=7,e=6,f=5,g=4,h=3,i=2,j=1}

assert_iter_order(
   sort(t),
   {{'a',10},{'b',9},{'c',8},{'d',7},{'e',6},{'f',5},{'g',4},{'h',3},{'i',2},{'j',1}})

assert_iter_order(
   sortv(t),
   {{'j',1},{'i',2},{'h',3},{'g',4},{'f',5},{'e',6},{'d',7},{'c',8},{'b',9},{'a',10}})


asserteq(difference({a = true, b = true},{a = true, b = true}),{})

-- no longer confused by false values ;)
asserteq(difference({v = false},{v = false}),{})

asserteq(difference({a = true},{b = true}),{a=true})

-- symmetric difference
asserteq(difference({a = true},{b = true},true),{a=true,b=true})

--basic index_map test
asserteq(index_map({10,20,30}), {[10]=1,[20]=2,[30]=3})
--test that repeated values return multiple indices
asserteq(index_map({10,20,30,30,30}), {[10]=1,[20]=2,[30]={3,4,5}})

-- Reduce
asserteq(tablex.reduce('-', {}, 2), 2)
asserteq(tablex.reduce('-', {}), nil)
asserteq(tablex.reduce('-', {1,2,3,4,5}), -13)
asserteq(tablex.reduce('-', {1,2,3,4,5}, 1), -14)


-- tablex.compare
do
  asserteq(tablex.compare({},{}, "=="), true)
  asserteq(tablex.compare({1,2,3}, {1,2,3}, "=="), true)
  asserteq(tablex.compare({1,"hello",3}, {1,2,3}, "=="), false)
  asserteq(tablex.compare(
      {1,2,3, hello = "world"},
      {1,2,3},
      function(v1, v2) return v1 == v2 end),
      true)  -- only compares the list part
end


-- tablex.rfind
do
  local rfind = tablex.rfind
  local lst = { "Rudolph", "the", "red-nose", "raindeer" }
  asserteq(rfind(lst, "Santa"), nil)
  asserteq(rfind(lst, "raindeer", -2), nil)
  asserteq(rfind(lst, "raindeer"), 4)
  asserteq(rfind(lst, "Rudolph"), 1)
  asserteq(rfind(lst, "the", -3), 2)
  asserteq(rfind(lst, "the", -30), nil)
  asserteq(rfind({10,10,10},10), 3)
end


-- tablex.find_if
do
  local fi = tablex.find_if
  local lst = { "Rudolph", true, false, 15 }
  asserteq({fi(lst, "==", "Rudolph")}, {1, true})
  asserteq({fi(lst, "==", true)}, {2, true})
  asserteq({fi(lst, "==", false)}, {3, true})
  asserteq({fi(lst, "==", 15)}, {4, true})

  local cmp = function(v1, v2) return v1 == v2 and v2 end
  asserteq({fi(lst, cmp, "Rudolph")}, {1, "Rudolph"})
  asserteq({fi(lst, cmp, true)}, {2, true})
  asserteq({fi(lst, cmp, false)}, {}) -- 'false' cannot be returned!
  asserteq({fi(lst, cmp, 15)}, {4, 15})
end


-- tablex.map_named_method
do
  local Car = {}
  Car.__index = Car
  function Car.new(car)
    return setmetatable(car or {}, Car)
  end
  Car.speed = 0
  function Car:faster(increase)
    self.speed = self.speed + (increase or 1)
    return self.speed
  end
  function Car:slower(self, decrease)
    self.speed = self.speed - (decrease or 1)
    return self.speed
  end

  local ferrari = Car.new{ name = "Ferrari" }
  local lamborghini = Car.new{ name = "Lamborghini", speed = 50 }
  local cars = { ferrari, lamborghini }

  asserteq(ferrari.speed, 0)
  asserteq(lamborghini.speed, 50)
  asserteq(tablex.map_named_method("faster", cars, 10), {10, 60})
  asserteq(ferrari.speed, 10)
  asserteq(lamborghini.speed, 60)

end


-- tablex.foreach
do
  local lst = { "one", "two", "three", hello = "world" }
  tablex.foreach(lst, function(v, k, sep)
    lst[k] = tostring(k) .. sep .. v
  end, " = ")
  asserteq(lst, {"1 = one", "2 = two", "3 = three", hello = "hello = world"})
end


-- tablex.foreachi
do
  local lst = { "one", "two", "three", hello = "world" }
  tablex.foreachi(lst, function(v, k, sep)
    lst[k] = tostring(k) .. sep .. v
  end, " = ")
  asserteq(lst, {"1 = one", "2 = two", "3 = three", hello = "world"})
end


-- tablex.new
asserteq(tablex.new(3, "hi"), { "hi", "hi", "hi" })


-- tablex.search
do
  local t = {
    penlight = {
      battery = {
        type = "AA",
        capacity = "1500mah",
      },
    },
    hello = {
      world = {
        also = "AA"
      }
    }
  }
  asserteq(tablex.search(t, "1500mah"), "penlight.battery.capacity")
  asserteq(tablex.search(t, "AA", {t.penlight} ), "hello.world.also")
  asserteq(tablex.search(t, "xxx"), nil)
end


-- tablex.readonly
do
  local ro = tablex.readonly { 1,2,3, hello = "world" }
  asserteq(pcall(function() ro.hello = "hi there" end), false)
  asserteq(getmetatable(ro), false)

  if not utils.lua51 then
    asserteq(#ro, 3)

    local r = {}
    for k,v in pairs(ro) do r[k] = v end
    asserteq(r, { 1,2,3, hello = "world" })

    r = {}
    for k,v in ipairs(ro) do r[k] = v end
    asserteq(r, { 1,2,3 })
  end
end
