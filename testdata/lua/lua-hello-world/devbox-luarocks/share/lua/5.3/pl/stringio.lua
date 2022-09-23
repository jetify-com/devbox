--- Reading and writing strings using file-like objects. <br>
--
--    f = stringio.open(text)
--    l1 = f:read()  -- read first line
--    n,m = f:read ('*n','*n') -- read two numbers
--    for line in f:lines() do print(line) end -- iterate over all lines
--    f = stringio.create()
--    f:write('hello')
--    f:write('dolly')
--    assert(f:value(),'hellodolly')
--
-- See  @{03-strings.md.File_style_I_O_on_Strings|the Guide}.
-- @module pl.stringio

local unpack = rawget(_G,'unpack') or rawget(table,'unpack')
local tonumber = tonumber
local concat,append = table.concat,table.insert

local stringio = {}

-- Writer class
local SW = {}
SW.__index = SW

local function xwrite(self,...)
    local args = {...} --arguments may not be nil!
    for i = 1, #args do
        append(self.tbl,args[i])
    end
end

function SW:write(arg1,arg2,...)
    if arg2 then
        xwrite(self,arg1,arg2,...)
    else
        append(self.tbl,arg1)
    end
end

function SW:writef(fmt,...)
    self:write(fmt:format(...))
end

function SW:value()
    return concat(self.tbl)
end

function SW:__tostring()
    return self:value()
end

function SW:close() -- for compatibility only
end

function SW:seek()
end

-- Reader class
local SR = {}
SR.__index = SR

function SR:_read(fmt)
    local i,str = self.i,self.str
    local sz = #str
    if i > sz then return nil end
    local res
    if fmt == '*l' or fmt == '*L' then
        local idx = str:find('\n',i) or (sz+1)
        res = str:sub(i,fmt == '*l' and idx-1 or idx)
        self.i = idx+1
    elseif fmt == '*a' then
        res = str:sub(i)
        self.i = sz
    elseif fmt == '*n' then
        local _,i2,idx
        _,idx = str:find ('%s*%d+',i)
        _,i2 = str:find ('^%.%d+',idx+1)
        if i2 then idx = i2 end
        _,i2 = str:find ('^[eE][%+%-]*%d+',idx+1)
        if i2 then idx = i2 end
        local val = str:sub(i,idx)
        res = tonumber(val)
        self.i = idx+1
    elseif type(fmt) == 'number' then
        res = str:sub(i,i+fmt-1)
        self.i = i + fmt
    else
        error("bad read format",2)
    end
    return res
end

function SR:read(...)
    if select('#',...) == 0 then
        return self:_read('*l')
    else
        local res, fmts = {},{...}
        for i = 1, #fmts do
            res[i] = self:_read(fmts[i])
        end
        return unpack(res)
    end
end

function SR:seek(whence,offset)
    local base
    whence = whence or 'cur'
    offset = offset or 0
    if whence == 'set' then
        base = 1
    elseif whence == 'cur' then
        base = self.i
    elseif whence == 'end' then
        base = #self.str
    end
    self.i = base + offset
    return self.i
end

function SR:lines(...)
    local n, args = select('#',...)
    if n > 0 then
        args = {...}
    end
    return function()
        if n == 0 then
            return self:_read '*l'
        else
            return self:read(unpack(args))
        end
    end
end

function SR:close() -- for compatibility only
end

--- create a file-like object which can be used to construct a string.
-- The resulting object has an extra `value()` method for
-- retrieving the string value.  Implements `file:write`, `file:seek`, `file:lines`,
-- plus an extra `writef` method which works like `utils.printf`.
-- @usage f = create(); f:write('hello, dolly\n'); print(f:value())
function stringio.create()
    return setmetatable({tbl={}},SW)
end

--- create a file-like object for reading from a given string.
-- Implements `file:read`.
-- @string s The input string.
-- @usage fs = open '20 10'; x,y = f:read ('*n','*n'); assert(x == 20 and y == 10)
function stringio.open(s)
    return setmetatable({str=s,i=1},SR)
end

function stringio.lines(s,...)
    return stringio.open(s):lines(...)
end

return stringio
