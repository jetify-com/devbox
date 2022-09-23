--- testing Lua 5.1/5.2 compatibility functions
-- these are global side-effects of pl.utils
local utils = require 'pl.utils'
local test = require 'pl.test'
local asserteq = test.asserteq
local _,lua = require 'pl.app'. lua()
local setfenv,getfenv = utils.setfenv, utils.getfenv

-- utils.execute is a compromise between 5.1 and 5.2 for os.execute changes
-- can we call Lua ?
local ok,code = utils.execute(lua..' -v')
asserteq(ok,true)
asserteq(code,0)
-- does it return false when it fails ?
asserteq(utils.execute('most-likely-nonexistent-command'),false)

-- table.pack is defined for 5.1
local t = table.pack(1,nil,'hello')
asserteq(t.n,3)
assert(t[1] == 1 and t[3] == 'hello')

-- unpack is not globally available for 5.2 unless in compat mode.
-- But utils.unpack is always defined.
local a,b = utils.unpack{10,'wow'}
assert(a == 10 and b == 'wow')

-- utils.load() is Lua 5.2 style
chunk = utils.load('return x+y','tmp','t',{x=1,y=2})
asserteq(chunk(),3)

-- can only load a binary chunk if the mode permits!
local f = string.dump(function() end)
local res,err = utils.load(f,'tmp','t')
test.assertmatch(err,'attempt to load')

-- package.searchpath for Lua 5.1
-- nota bene: depends on ./?.lua being in the package.path!
-- So we hack it if not found
if not package.path:find '.[/\\]%?' then
    package.path = './?.lua;'..package.path
end

asserteq(
    package.searchpath('tests.test-utils3',package.path):gsub('\\','/'),
    './tests/test-utils3.lua'
)

-- testing getfenv and setfenv for both interpreters

function test()
    return X + Y + Z
end

t = {X = 1, Y = 2, Z = 3}

setfenv(test,t)

assert(test(),6)

t.X = 10

assert(test(),15)

local getfenv,_G = getfenv,_G

function test2()
    local env = {x=2}
    setfenv(1,env)
    asserteq(getfenv(1),env)
    asserteq(x,2)
end

test2()



