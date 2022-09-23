---- testing types
local types = require 'pl.types'
local asserteq = require 'pl.test'.asserteq
local List = require 'pl.List'

local list = List()
local array = {10,20,30}
local map = {one=1,two=2}

-- extended type() function
asserteq(types.type(array),'table')
asserteq(types.type('hello'),'string')
-- knows about Lua file objects
asserteq(types.type(io.stdin),'file')
local f = io.open("tests/test-types.lua")
asserteq(types.type(f),'file')
f:close()
-- and class names
asserteq(types.type(list),'List')
-- tables with unknown metatable
asserteq(types.type(setmetatable({},{})), "unknown table")
-- userdata with unknown metatable
if newproxy then
    asserteq(types.type(newproxy(true)), "unknown userdata")
end

asserteq(types.is_integer(10),true)
asserteq(types.is_integer(10.1),false)
asserteq(types.is_integer(-10),true)
asserteq(types.is_integer(-10.1),false)
-- do note that for Lua < 5.3, 10.0 is the same as 10; an integer.

asserteq(types.is_callable(asserteq),true)
asserteq(types.is_callable(List),true)

asserteq(types.is_indexable(array),true)
asserteq(types.is_indexable('hello'),nil)
asserteq(types.is_indexable(10),nil)
if newproxy then
    local v = newproxy(true)
    local mt = getmetatable(v)
    mt.__len = true
    mt.__index = true
    asserteq(types.is_indexable(v), true)
end
if newproxy then
    local v = newproxy(true)
    asserteq(types.is_indexable(v), nil)
end

asserteq(types.is_iterable(array),true)
asserteq(types.is_iterable(true),nil)
asserteq(types.is_iterable(42),nil)
asserteq(types.is_iterable("array"),nil)
if newproxy then
    local v = newproxy(true)
    local mt = getmetatable(v)
    mt.__pairs = true
    asserteq(types.is_iterable(v), true)
end
if newproxy then
    local v = newproxy(true)
    asserteq(types.is_iterable(v), nil)
end

asserteq(types.is_writeable(array),true)
asserteq(types.is_writeable(true),nil)
asserteq(types.is_writeable(42),nil)
asserteq(types.is_writeable("array"),nil)
if newproxy then
    local v = newproxy(true)
    local mt = getmetatable(v)
    mt.__newindex = true
    asserteq(types.is_writeable(v), true)
end
if newproxy then
    local v = newproxy(true)
    asserteq(types.is_writeable(v), nil)
end

asserteq(types.is_empty(nil),true)
asserteq(types.is_empty({}),true)
asserteq(types.is_empty({[false] = false}),false)
asserteq(types.is_empty(""),true)
asserteq(types.is_empty("   ",true),true)
asserteq(types.is_empty("   "),false)
asserteq(types.is_empty(true),true)
-- Numbers
asserteq(types.is_empty(0), true)
asserteq(types.is_empty(20), true)
-- Booleans
asserteq(types.is_empty(false), true)
asserteq(types.is_empty(true), true)
-- Functions
asserteq(types.is_empty(print), true)
-- Userdata
--asserteq(types.is_empty(newproxy()), true)  --newproxy was removed in Lua 5.2

-- a more relaxed kind of truthiness....
asserteq(types.to_bool('yes'),true)
asserteq(types.to_bool('true'),true)
asserteq(types.to_bool('y'),true)
asserteq(types.to_bool('t'),true)
asserteq(types.to_bool('YES'),true)
asserteq(types.to_bool('1'),true)
asserteq(types.to_bool('no'),false)
asserteq(types.to_bool('false'),false)
asserteq(types.to_bool('n'),false)
asserteq(types.to_bool('f'),false)
asserteq(types.to_bool('NO'),false)
asserteq(types.to_bool('0'),false)
asserteq(types.to_bool(1),true)
asserteq(types.to_bool(0),false)
local de_fr = { 'ja', 'oui' }
asserteq(types.to_bool('ja', de_fr),true)
asserteq(types.to_bool('OUI', de_fr),true)
local t_e = {}
local t_ne = { "not empty" }
asserteq(types.to_bool(t_e,nil,false),false)
asserteq(types.to_bool(t_e,nil,true),false)
asserteq(types.to_bool(t_ne,nil,false),false)
asserteq(types.to_bool(t_ne,nil,true),true)
asserteq(types.to_bool(coroutine.create(function() end),nil,true),true)
asserteq(types.to_bool(coroutine.create(function() end),nil,false),false)
