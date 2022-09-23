local sip = require 'pl.sip'
local tablex = require 'pl.tablex'
local test = require 'pl.test'

local function check(pat,line,tbl)
    local parms = {}
    if type(pat) == 'string' then
        pat = sip.compile(pat)
    end
    if pat(line,parms) then
        test.asserteq(parms,tbl)
    else -- only should happen if we're passed a nil!
        assert(tbl == nil)
    end
end

local c = sip.compile('ref=$S{file}:$d{line}')
check(c,'ref=bonzo:23',{file='bonzo',line=23})
check(c,'here we go ref=c:\\bonzo\\dog.txt:53',{file='c:\\bonzo\\dog.txt',line=53})
check(c,'here is a line ref=xxxx:xx',nil)

c = sip.compile('($i{x},$i{y},$i{z})')
check(c,'(10,20,30)',{x=10,y=20,z=30})
check(c,'  (+233,+99,-40) ',{x=233,y=99,z=-40})

local pat = '$v{name} = $q{str}'
--assert(sip.create_pattern(pat) == [[([%a_][%w_]*)%s*=%s*(["'])(.-)%2]])
local m = sip.compile(pat)

check(m,'a = "hello"',{name='a',str='hello'})
check(m,"a = 'hello'",{name='a',str='hello'})
check(m,'_fred="some text"',{name='_fred',str='some text'})

-- some cases broken in 0.6b release
check('$v is $v','bonzo is dog for sure',{'bonzo','dog'})
check('$v is $','bonzo is dog for sure',{'bonzo','dog for sure'})

-- spaces
check('$v $d','age 23',{'age',23})
check('$v $d','age  23',{'age',23})
check('$v $d','age23') -- the space is 'imcompressible'
check('a b c $r', 'a bc d')
check('a b c $r', 'a b c d',{'d'})

-- the spaces in this pattern, however, are compressible.
check('$v = $d','age=23',{'age',23})

-- patterns without patterns
check('just a string', 'just a string', {})
check('just a string', 'not that string')

local months={"Jan","Feb","Mar","Apr","May","Jun","Jul","Aug","Sep","Oct","Nov","Dec"}

local function adjust_year(res)
    if res.year < 100 then
        if res.year < 70 then
            res.year = res.year + 2000
        else
            res.year = res.year + 1900
        end
    end
end

local shortdate = sip.compile('$d{day}/$d{month}/$d{year}')
local longdate = sip.compile('$d{day} $v{month} $d{year}')
local isodate = sip.compile('$d{year}-$d{month}-$d{day}')

local function dcheck (d1,d2)
    adjust_year(d1)
    test.asserteq(d1, d2)
end

local function dates(str,tbl)
    local res = {}
    if shortdate(str,res) then
        dcheck(res,tbl)
    elseif isodate(str,res) then
        dcheck(res,tbl)
    elseif longdate(str,res) then
        res.month = tablex.find(months,res.month)
        dcheck(res,tbl)
    else
        assert(tbl == nil)
    end
end

dates ('10/12/2007',{year=2007,month=12,day=10})
dates ('2006-03-01',{year=2006,month=3,day=1})
dates ('25/07/05',{year=2005,month=7,day=25})
dates ('20 Mar 1959',{year=1959,month=3,day=20})

local sio = require 'pl.stringio'
local lines = [[
dodge much amazement
kitteh cheezburger
]]
sip.read(sio.open(lines),{
    {'dodge $',function(rest) test.asserteq(rest,'much amazement') end},
    {'kitteh $',function(rest) test.asserteq(rest,'cheezburger') end}
})





