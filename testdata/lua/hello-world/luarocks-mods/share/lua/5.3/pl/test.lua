--- Useful test utilities.
--
--    test.asserteq({1,2},{1,2}) -- can compare tables
--    test.asserteq(1.2,1.19,0.02) -- compare FP numbers within precision
--    T = test.tuple -- used for comparing multiple results
--    test.asserteq(T(string.find(" me","me")),T(2,3))
--
-- Dependencies: `pl.utils`, `pl.tablex`, `pl.pretty`, `pl.path`, `debug`
-- @module pl.test

local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local pretty = require 'pl.pretty'
local path = require 'pl.path'
local type,unpack,pack = type,utils.unpack,utils.pack
local clock = os.clock
local debug = require 'debug'
local io = io

local function dump(x)
    if type(x) == 'table' and not (getmetatable(x) and getmetatable(x).__tostring) then
        return pretty.write(x,' ',true)
    elseif type(x) == 'string' then
        return '"'..x..'"'
    else
        return tostring(x)
    end
end

local test = {}

---- error handling for test results.
-- By default, this writes to stderr and exits the program.
-- Re-define this function to raise an error and/or redirect output
function test.error_handler(file,line,got_text, needed_text,msg)
    local err = io.stderr
    err:write(path.basename(file)..':'..line..': assertion failed\n')
    err:write("got:\t",got_text,'\n')
    err:write("needed:\t",needed_text,'\n')
    utils.quit(1,msg or "these values were not equal")
end

local function complain (x,y,msg,where)
    local i = debug.getinfo(3 + (where or 0))
    test.error_handler(i.short_src,i.currentline,dump(x),dump(y),msg)
end

--- general test complain message.
-- Useful for composing new test functions (see tests/tablex.lua for an example)
-- @param x a value
-- @param y value to compare first value against
-- @param msg message
-- @param where extra level offset for errors
-- @function complain
test.complain = complain

--- like assert, except takes two arguments that must be equal and can be tables.
-- If they are plain tables, it will use tablex.deepcompare.
-- @param x any value
-- @param y a value equal to x
-- @param eps an optional tolerance for numerical comparisons
-- @param where extra level offset
function test.asserteq (x,y,eps,where)
    local res = x == y
    if not res then
        res = tablex.deepcompare(x,y,true,eps)
    end
    if not res then
        complain(x,y,nil,where)
    end
end

--- assert that the first string matches the second.
-- @param s1 a string
-- @param s2 a string
-- @param where extra level offset
function test.assertmatch (s1,s2,where)
    if not s1:match(s2) then
        complain (s1,s2,"these strings did not match",where)
    end
end

--- assert that the function raises a particular error.
-- @param fn a function or a table of the form {function,arg1,...}
-- @param e a string to match the error against
-- @param where extra level offset
function test.assertraise(fn,e,where)
    local ok, err
    if type(fn) == 'table' then
        ok, err = pcall(unpack(fn))
    else
        ok, err = pcall(fn)
    end
    if ok or err:match(e)==nil then
        complain (err,e,"these errors did not match",where)
    end
end

--- a version of asserteq that takes two pairs of values.
-- <code>x1==y1 and x2==y2</code> must be true. Useful for functions that naturally
-- return two values.
-- @param x1 any value
-- @param x2 any value
-- @param y1 any value
-- @param y2 any value
-- @param where extra level offset
function test.asserteq2 (x1,x2,y1,y2,where)
    if x1 ~= y1 then complain(x1,y1,nil,where) end
    if x2 ~= y2 then complain(x2,y2,nil,where) end
end

-- tuple type --

local tuple_mt = {
    unpack = unpack
}
tuple_mt.__index = tuple_mt

function tuple_mt.__tostring(self)
    local ts = {}
    for i=1, self.n do
        local s = self[i]
        ts[i] = type(s) == 'string' and ('%q'):format(s) or tostring(s)
    end
    return 'tuple(' .. table.concat(ts, ', ') .. ')'
end

function tuple_mt.__eq(a, b)
    if a.n ~= b.n then return false end
    for i=1, a.n do
        if a[i] ~= b[i] then return false end
    end
    return true
end

function tuple_mt.__len(self)
    return self.n
end

--- encode an arbitrary argument list as a tuple.
-- This can be used to compare to other argument lists, which is
-- very useful for testing functions which return a number of values.
-- Unlike regular array-like tables ('sequences') they may contain nils.
-- Tuples understand equality and know how to print themselves out.
-- The # operator is defined to be the size, irrespecive of any nils,
-- and there is an `unpack` method.
-- @usage asserteq(tuple( ('ab'):find 'a'), tuple(1,1))
function test.tuple(...)
    return setmetatable(pack(...), tuple_mt)
end

--- Time a function. Call the function a given number of times, and report the number of seconds taken,
-- together with a message.  Any extra arguments will be passed to the function.
-- @string msg a descriptive message
-- @int n number of times to call the function
-- @func fun the function
-- @param ... optional arguments to fun
function test.timer(msg,n,fun,...)
    local start = clock()
    for i = 1,n do fun(...) end
    utils.printf("%s: took %7.2f sec\n",msg,clock()-start)
end

return test
