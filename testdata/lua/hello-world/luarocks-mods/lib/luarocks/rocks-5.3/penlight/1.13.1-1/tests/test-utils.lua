local utils = require 'pl.utils'
local path = require 'pl.path'
local test = require 'pl.test'
local asserteq, T = test.asserteq, test.tuple


local function quote(s)
    if utils.is_windows then
        return '"'..s..'"'
    else
        return "'"..s.."'"
    end
end

-- construct command to run external lua, we need to to be able to run some
-- tests on the same lua engine, but also need to pass on the LuaCov flag
-- if it was used, to make sure we report the proper coverage.
local cmd = "-e "
do
    local i = 0
    while arg[i-1] do
      local a = arg[i-1]
      if a:find("package%.path") and a:sub(1,1) ~= "'" then
        a = quote(a)
      end
      cmd = a .. " " .. cmd
      i = i - 1
    end
end


--- quitting
do
    local luacode = quote("require([[pl.utils]]).quit([[hello world]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    if utils.is_windows then
        asserteq(code, -1)
    else
        asserteq(code, 255)
    end
    asserteq(stdout, "")
    asserteq(stderr, "hello world\n")

    local luacode = quote("require([[pl.utils]]).quit(2, [[hello world]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    asserteq(code, 2)
    asserteq(stdout, "")
    asserteq(stderr, "hello world\n")

    local luacode = quote("require([[pl.utils]]).quit(2, [[hello %s]], 42)")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    asserteq(code, 2)
    asserteq(stdout, "")
    asserteq(stderr, "hello 42\n")

    local luacode = quote("require([[pl.utils]]).quit(2)")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    asserteq(code, 2)
    asserteq(stdout, "")
    asserteq(stderr, "")
end

----- importing module tables wholesale ---
utils.import(math)
asserteq(type(sin),"function")
asserteq(type(abs),"function")

--- useful patterns
local P = utils.patterns
asserteq(("+0.1e10"):match(P.FLOAT) ~= nil, true)
asserteq(("-23430"):match(P.INTEGER) ~= nil, true)
asserteq(("my_little_pony99"):match(P.IDEN) ~= nil, true)

--- escaping magic chars
local escape = utils.escape
asserteq(escape '[a]','%[a%]')
asserteq(escape '$(bonzo)','%$%(bonzo%)')

--- choose
asserteq(utils.choose(true, 1, 2), 1)
asserteq(utils.choose(false, 1, 2), 2)

--- splitting strings ---
local split = utils.split
asserteq(split("hello dolly"),{"hello","dolly"})
asserteq(split("hello,dolly",","),{"hello","dolly"})
asserteq(split("hello,dolly,",","),{"hello","dolly"})

local first,second = utils.splitv("hello:dolly",":")
asserteq(T(first,second),T("hello","dolly"))
local first,second = utils.splitv("hello:dolly:parton",":", false, 2)
asserteq(T(first,second),T("hello","dolly:parton"))
local first,second,third = utils.splitv("hello=dolly:parton","[:=]")
asserteq(T(first,second,third),T("hello","dolly","parton"))
local first,second = utils.splitv("hello=dolly:parton","[:=]", false, 2)
asserteq(T(first,second),T("hello","dolly:parton"))

----- table of values to table of strings
asserteq(utils.array_tostring{1,2,3},{"1","2","3"})
-- writing into existing table
local tmp = {}
utils.array_tostring({1,2,3},tmp)
asserteq(tmp,{"1","2","3"})

--- memoizing a function
local kount = 0
local f = utils.memoize(function(x)
    kount = kount + 1
    return x*x
end)
asserteq(f(2),4)
asserteq(f(10),100)
asserteq(f(2),4)
-- actual function only called twice
asserteq(kount,2)

-- string lambdas
local L = utils.string_lambda
local g = L"|x| x:sub(1,1)"
asserteq(g("hello"),"h")

local f = L"|x,y| x - y"
asserteq(f(10,2),8)

-- alternative form for _one_ argument
asserteq(L("2 * _")(4), 8)

local List = require 'pl.List'
local ls = List{10,20,30}

-- string lambdas can be used throughout Penlight
asserteq(ls:map"_+1", {11,21,31})

-- because they use this common function
local function test_fn_arg(f)
    f = utils.function_arg(1,f)
    asserteq(f(10),11)
end

test_fn_arg (function (x) return x + 1 end)
test_fn_arg  '_ + 1'
test.assertraise(function() test_fn_arg {} end, 'not a callable object')
test.assertraise(function() test_fn_arg (0) end, 'must be callable')

-- partial application

local f1 = utils.bind1(f,10)
asserteq(f1(2), 8)

local f2 = utils.bind2(f,2)
asserteq(f2(10), 8)

--- extended type checking

local is_type = utils.is_type
-- anything without a metatable works as regular type() function
asserteq(is_type("one","string"),true)
asserteq(is_type({},"table"),true)

-- but otherwise the type of an object is considered to be its metatable
asserteq(is_type(ls,List),true)

-- compatibility functions
local chunk = utils.load 'return 42'
asserteq(chunk(),42)

chunk = utils.load 'a = 42'
chunk()
asserteq(a,42)

local t = {}
chunk = utils.load ('b = 42','<str>','t',t)
chunk()
asserteq(t.b,42)

chunk,err = utils.load ('a = ?','<str>')
assert(err,[[[string "<str>"]:1: unexpected symbol near '?']])

asserteq(utils.quote_arg("foo"), [[foo]])
if path.is_windows then
    asserteq(utils.quote_arg(""), '^"^"')
    asserteq(utils.quote_arg('"'), '^"')
    asserteq(utils.quote_arg([[ \]]), [[^" \\^"]])
    asserteq(utils.quote_arg([[foo\\ bar\\" baz\]]), [[^"foo\\ bar\\\\\^" baz\\^"]])
    asserteq(utils.quote_arg("%path% ^^!()"), [[^"^%path^% ^^^^^!()^"]])
else
    asserteq(utils.quote_arg(""), "''")
    asserteq(utils.quote_arg("'"), [[''\''']])
    asserteq(utils.quote_arg([['a\'b]]), [[''\''a\'\''b']])
end

-- packing and unpacking arguments in a nil-safe way
local t = utils.pack(nil, nil, "hello", nil)
asserteq(t.n, 4) -- the last nil does count as an argument

local arg1, arg2, arg3, arg4 = utils.unpack(t)
assert(arg1 == nil)
assert(arg2 == nil)
asserteq("hello", arg3)
assert(arg4 == nil)


-- Assert arguments assert_arg
local ok, err = pcall(function()
    utils.assert_arg(4,'!@#$%^&*','string',require("pl.path").isdir,'not a directory')
end)
asserteq(ok, false)
asserteq(err:match("(argument .+)$"), "argument 4: '!@#$%^&*' not a directory")

local ok, err = pcall(function()
    utils.assert_arg(1, "hello", "table")
end)
asserteq(ok, false)
asserteq(err:match("(argument .+)$"), "argument 1 expected a 'table', got a 'string'")

local ok, err = pcall(function()
    return utils.assert_arg(1, "hello", "string")
end)
asserteq(ok, true)
asserteq(err, "hello")

-- assert_string
local success, err = pcall(utils.assert_string, 2, 5)
asserteq(success, false)
asserteq(err:match("(argument .+)$"), "argument 2 expected a 'string', got a 'number'")

local x = utils.assert_string(2, "5")
asserteq(x, "5")


do
    -- printf -- without template
    local luacode = quote("require([[pl.utils]]).printf([[hello world]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, true)
    asserteq(code, 0)
    asserteq(stdout, "hello world")
    asserteq(stderr, "")

    -- printf -- with template
    local luacode = quote("require([[pl.utils]]).printf([[hello %s]], [[world]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, true)
    asserteq(code, 0)
    asserteq(stdout, "hello world")
    asserteq(stderr, "")

    -- printf -- with bad template
    local luacode = quote("require([[pl.utils]]).printf(42)")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    asserteq(code, 1)
    asserteq(stdout, "")
    assert(stderr:find("argument 1 expected a 'string', got a 'number'"))
end

do
    -- on_error, raise  -- default
    utils.on_error("default")
    local ok, err = utils.raise("some error")
    asserteq(ok, nil)
    asserteq(err, "some error")
    local ok, err = pcall(utils.on_error, "bad one")
    asserteq(ok, false)
    asserteq(err, "Bad argument expected string; 'default', 'quit', or 'error'. Got 'bad one'")

    -- on_error, raise  -- error
    utils.on_error("error")
    local ok, err = pcall(utils.raise, "some error")
    asserteq(ok, false)
    asserteq(err, "some error")
    local ok, err = pcall(utils.on_error, "bad one")
    asserteq(ok, false)
    assert(err:find("Bad argument expected string; 'default', 'quit', or 'error'. Got 'bad one'"))

    -- on_error, raise  -- quit
    utils.on_error("quit")
    local luacode = quote("local u=require([[pl.utils]]) u.on_error([[quit]]) u.raise([[some error]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    if utils.is_windows then
        asserteq(code, -1)
    else
        asserteq(code, 255)
    end
    asserteq(stdout, "")
    asserteq(stderr, "some error\n")

    local luacode = quote("local u=require([[pl.utils]]) u.on_error([[quit]]) u.on_error([[bad one]])")
    local success, code, stdout, stderr = utils.executeex(cmd..luacode)
    asserteq(success, false)
    if utils.is_windows then
        asserteq(code, -1)
    else
        asserteq(code, 255)
    end
    asserteq(stdout, "")
    asserteq(stderr, "Bad argument expected string; 'default', 'quit', or 'error'. Got 'bad one'\n")

    utils.on_error("default") -- cleanup by restoring behaviour after on_error + raise tests
end

do
    -- readlines
    local f = utils.readlines("tests/test-utils.lua")
    asserteq(type(f), "table")
    local v = "some extraordinary string this is only in this file for test purposes so we can go and find it"
    local found = false
    for i, line in ipairs(f) do
      if line:find(v) then
        found = true
        break
      end
    end
    asserteq(found, true)
end
