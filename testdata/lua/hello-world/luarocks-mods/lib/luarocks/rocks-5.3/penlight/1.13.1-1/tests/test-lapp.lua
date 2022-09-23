
local test = require 'pl.test'
local lapp = require 'pl.lapp'
local utils = require 'pl.utils'
local tablex = require 'pl.tablex'
local path = require 'pl.path'
local normpath = path.normpath

local k = 1
function check (spec,args,match)
    local args = lapp(spec,args)
    for k,v in pairs(args) do
        if type(v) == 'userdata' then args[k]:close(); args[k] = '<file>' end
    end
    test.asserteq(args,match,nil,1)
end

-- force Lapp to throw an error, rather than just calling os.exit()
lapp.show_usage_error = 'throw'

function check_error(spec,args,msg)
    arg = args
    local ok,err = pcall(lapp,spec)
    test.assertmatch(err,msg)
end

local parmtest = [[
Testing 'array' parameter handling
    -o,--output... (string)
    -v...
]]


check (parmtest,{'-o','one'},{output={'one'},v={false}})
check (parmtest,{'-o','one','-v'},{output={'one'},v={true}})
check (parmtest,{'-o','one','-vv'},{output={'one'},v={true,true}})
check (parmtest,{'-o','one','-o','two'},{output={'one','two'},v={false}})


local simple = [[
Various flags and option types
    -p          A simple optional flag, defaults to false
    -q,--quiet  A simple flag with long name
    -o  (string)  A required option with argument
    <input> (default stdin)  Optional input file parameter...
]]

check(simple,
    {'-o','in'},
    {quiet=false,p=false,o='in',input='<file>', input_name="stdin"})

---- value of flag may be separated by '=' or ':'
check(simple,
    {'-o=in'},
    {quiet=false,p=false,o='in',input='<file>', input_name="stdin"})

check(simple,
    {'-o:in'},
    {quiet=false,p=false,o='in',input='<file>', input_name="stdin"})

-- Check lapp.callback.
local calls = {}
function lapp.callback(param, arg)
    table.insert(calls, {param, arg})
end
check(simple,
    {'-o','help','-q',normpath 'tests/test-lapp.lua'},
    {quiet=true,p=false,o='help',input='<file>',input_name=normpath 'tests/test-lapp.lua'})
test.asserteq(calls, {
    {'o', 'help'},
    {'quiet', '-q'},
    {'input', normpath 'tests/test-lapp.lua'}
})
lapp.callback = nil

local longs = [[
    --open (string)
]]

check(longs,{'--open','folder'},{open='folder'})

local long_file = [[
    --open (default stdin)
]]

check(long_file,{'--open',normpath 'tests/test-lapp.lua'},{open='<file>',open_name=normpath 'tests/test-lapp.lua'})

local extras1 = [[
    <files...> (string) A bunch of files
]]

check(extras1,{'one','two'},{files={'one','two'}})

-- any extra parameters go into the array part of the result
local extras2 = [[
    <file> (string) A file
]]

check(extras2,{'one','two'},{file='one','two'})

local extended = [[
    --foo (string default 1)
    -s,--speed (slow|medium|fast default medium)
    -n (1..10 default 1)
    -p print
    -v verbose
]]


check(extended,{},{foo='1',speed='medium',n=1,p=false,v=false})
check(extended,{'-pv'},{foo='1',speed='medium',n=1,p=true,v=true})
check(extended,{'--foo','2','-s','fast'},{foo='2',speed='fast',n=1,p=false,v=false})
check(extended,{'--foo=2','-s=fast','-n2'},{foo='2',speed='fast',n=2,p=false,v=false})

check_error(extended,{'--speed','massive'},"value 'massive' not in slow|medium|fast")

check_error(extended,{'-n','x'},"unable to convert to number: x")

check_error(extended,{'-n','12'},"n out of range")

local with_advanced_enum = [[
  -s  (test1|test2()|%a)
  -c  (1-2|2-3|cool[])
]]

check(with_advanced_enum,{"-s", "test2()", "-c", "1-2"},{s='test2()',c='1-2'})
check(with_advanced_enum,{"-s", "test2()", "-c", "2-3"},{s='test2()',c='2-3'})
check(with_advanced_enum,{"-s", "%a", "-c", "2-3"},{s='%a',c='2-3'})

local with_dashes = [[
  --first-dash  dash
  --second-dash dash also
]]

check(with_dashes,{'--first-dash'},{first_dash=true,second_dash=false})

-- optional parameters don't have to be set
local optional = [[
  -p (optional string)
]]

check(optional,{'-p', 'test'},{p='test'})
check(optional,{},{})

-- boolean flags may have a true default...
local false_flag = [[
    -g group results
    -f (default true) force result
]]

check (false_flag,{},{f=true,g=false})

check (false_flag,{'-g','-f'},{f=false,g=true})

-- '--' indicates end of parameter parsing
check (false_flag,{'-g','--'},{f=true,g=true})
check (false_flag,{'-g','--','-a','frodo'},{f=true,g=true; '-a','frodo'})



local default_file_flag = [[
    -f (file-out default stdout)
]]
check (default_file_flag,{},{f="<file>", f_name = "stdout"})



local numbered_pos_args = [[
    <arg1>     (string)
    <arg2>     (string)
    <3arg3>    (string)
]]
check (numbered_pos_args,{"1", "2", "3"},{arg1="1", arg2 = "2", _arg3 = "3"})


local addtype = [[
  -l (intlist) List of items
]]

-- defining a custom type
lapp.add_type('intlist',
              function(x)
                 return tablex.imap(tonumber, utils.split(x, '%s*,%s*'))
              end,
              function(x)
                 for _,v in ipairs(x) do
                    lapp.assert(math.ceil(v) == v,'not an integer!')
                 end
              end)

check(addtype,{'-l', '1,2,3'},{l={1,2,3}})

check_error(addtype,{'-l', '1.5,2,3'},"not an integer!")

-- short flags may be immediately followed by their value
-- (previously only true for numerical values)
local short_args = [[
    -n (default 10)
    -I,--include (string)
]]

check(short_args,{'-Ifrodo','-n5'},{include='frodo',n=5})
check(short_args,{'-I/usr/local/lua/5.1'},{include='/usr/local/lua/5.1',n=10})

-- ok, introducing _slack_ mode ;)
-- 'short' flags may have multiple characters! (this is otherwise an error)
-- Note that in _any case_ flags may contain hyphens, but these are turned
-- into underscores for convenience.
lapp.slack = true
local spec = [[
Does some calculations
   -vs,--video-set              (string)             Use the German road sign dataset
   -w,--width              (default 256)        Width of the video
   -h,--height             (default 144)        Height of the video
   -t,--time               (default 10)         Seconds of video to process
   -sk,--seek               (default 0)          Seek number of seconds
   -dbg                   Debug!
]]

test.asserteq(lapp(spec,{'-vs',200,'-sk',1}),{
  video_set = 200,
  time = 10,
  height = 144,
  seek = 1,
  dbg = false,
  width = 256
})

