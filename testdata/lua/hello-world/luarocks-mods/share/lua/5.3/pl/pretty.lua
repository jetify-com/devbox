--- Pretty-printing Lua tables.
-- Also provides a sandboxed Lua table reader and
-- a function to present large numbers in human-friendly format.
--
-- Dependencies: `pl.utils`, `pl.lexer`, `pl.stringx`, `debug`
-- @module pl.pretty

local append = table.insert
local concat = table.concat
local mfloor, mhuge = math.floor, math.huge
local mtype = math.type
local utils = require 'pl.utils'
local lexer = require 'pl.lexer'
local debug = require 'debug'
local quote_string = require'pl.stringx'.quote_string
local assert_arg = utils.assert_arg

local original_tostring = tostring

-- Patch tostring to format numbers with better precision
-- and to produce cross-platform results for
-- infinite values and NaN.
local function tostring(value)
    if type(value) ~= "number" then
        return original_tostring(value)
    elseif value ~= value then
        return "NaN"
    elseif value == mhuge then
        return "Inf"
    elseif value == -mhuge then
        return "-Inf"
    elseif (_VERSION ~= "Lua 5.3" or mtype(value) == "integer") and mfloor(value) == value then
        return ("%d"):format(value)
    else
        local res = ("%.14g"):format(value)
        if _VERSION == "Lua 5.3" and mtype(value) == "float" and not res:find("%.") then
            -- Number is internally a float but looks like an integer.
            -- Insert ".0" after first run of digits.
            res = res:gsub("%d+", "%0.0", 1)
        end
        return res
    end
end

local pretty = {}

local function save_global_env()
    local env = {}
    env.hook, env.mask, env.count = debug.gethook()

    -- env.hook is "external hook" if is a C hook function
    if env.hook~="external hook" then
        debug.sethook()
    end

    env.string_mt = getmetatable("")
    debug.setmetatable("", nil)
    return env
end

local function restore_global_env(env)
    if env then
        debug.setmetatable("", env.string_mt)
        if env.hook~="external hook" then
            debug.sethook(env.hook, env.mask, env.count)
        end
    end
end

--- Read a string representation of a Lua table.
-- This function loads and runs the string as Lua code, but bails out
-- if it contains a function definition.
-- Loaded string is executed in an empty environment.
-- @string s string to read in `{...}` format, possibly with some whitespace
-- before or after the curly braces. A single line comment may be present
-- at the beginning.
-- @return a table in case of success.
-- If loading the string failed, return `nil` and error message.
-- If executing loaded string failed, return `nil` and the error it raised.
function pretty.read(s)
    assert_arg(1,s,'string')
    if s:find '^%s*%-%-' then -- may start with a comment..
        s = s:gsub('%-%-.-\n','')
    end
    if not s:find '^%s*{' then return nil,"not a Lua table" end
    if s:find '[^\'"%w_]function[^\'"%w_]' then
        local tok = lexer.lua(s)
        for t,v in tok do
            if t == 'keyword' and v == 'function' then
                return nil,"cannot have functions in table definition"
            end
        end
    end
    s = 'return '..s
    local chunk,err = utils.load(s,'tbl','t',{})
    if not chunk then return nil,err end
    local global_env = save_global_env()
    local ok,ret = pcall(chunk)
    restore_global_env(global_env)
    if ok then return ret
    else
        return nil,ret
    end
end

--- Read a Lua chunk.
-- @string s Lua code.
-- @tab[opt] env environment used to run the code, empty by default.
-- @bool[opt] paranoid abort loading if any looping constructs a found in the code
-- and disable string methods.
-- @return the environment in case of success or `nil` and syntax or runtime error
-- if something went wrong.
function pretty.load (s, env, paranoid)
    env = env or {}
    if paranoid then
        local tok = lexer.lua(s)
        for t,v in tok do
            if t == 'keyword'
                and (v == 'for' or v == 'repeat' or v == 'function' or v == 'goto')
            then
                return nil,"looping not allowed"
            end
        end
    end
    local chunk,err = utils.load(s,'tbl','t',env)
    if not chunk then return nil,err end
    local global_env = paranoid and save_global_env()
    local ok,err = pcall(chunk)
    restore_global_env(global_env)
    if not ok then return nil,err end
    return env
end

local function quote_if_necessary (v)
    if not v then return ''
    else
        --AAS
        if v:find ' ' then v = quote_string(v) end
    end
    return v
end

local keywords

local function is_identifier (s)
    return type(s) == 'string' and s:find('^[%a_][%w_]*$') and not keywords[s]
end

local function quote (s)
    if type(s) == 'table' then
        return pretty.write(s,'')
    else
        --AAS
        return quote_string(s)-- ('%q'):format(tostring(s))
    end
end

local function index (numkey,key)
    --AAS
    if not numkey then
        key = quote(key)
         key = key:find("^%[") and (" " .. key .. " ") or key
    end
    return '['..key..']'
end


--- Create a string representation of a Lua table.
-- This function never fails, but may complain by returning an
-- extra value. Normally puts out one item per line, using
-- the provided indent; set the second parameter to an empty string
-- if you want output on one line.
--
-- *NOTE:* this is NOT a serialization function, not a full blown
-- debug function. Checkout out respectively the
-- [serpent](https://github.com/pkulchenko/serpent)
-- or [inspect](https://github.com/kikito/inspect.lua)
-- Lua modules for that if you need them.
-- @tab tbl Table to serialize to a string.
-- @string[opt] space The indent to use.
-- Defaults to two spaces; pass an empty string for no indentation.
-- @bool[opt] not_clever Pass `true` for plain output, e.g `{['key']=1}`.
-- Defaults to `false`.
-- @return a string
-- @return an optional error message
function pretty.write (tbl,space,not_clever)
    if type(tbl) ~= 'table' then
        local res = tostring(tbl)
        if type(tbl) == 'string' then return quote(tbl) end
        return res, 'not a table'
    end
    if not keywords then
        keywords = lexer.get_keywords()
    end
    local set = ' = '
    if space == '' then set = '=' end
    space = space or '  '
    local lines = {}
    local line = ''
    local tables = {}


    local function put(s)
        if #s > 0 then
            line = line..s
        end
    end

    local function putln (s)
        if #line > 0 then
            line = line..s
            append(lines,line)
            line = ''
        else
            append(lines,s)
        end
    end

    local function eat_last_comma ()
        local n = #lines
        local lastch = lines[n]:sub(-1,-1)
        if lastch == ',' then
            lines[n] = lines[n]:sub(1,-2)
        end
    end


    -- safe versions for iterators since 5.3+ honors metamethods that can throw
    -- errors
    local ipairs = function(t)
        local i = 0
        local ok, v
        local getter = function() return t[i] end
        return function()
                i = i + 1
                ok, v = pcall(getter)
                if v == nil or not ok then return end
                return i, t[i]
            end
    end
    local pairs = function(t)
        local k, v, ok
        local getter = function() return next(t, k) end
        return function()
                ok, k, v = pcall(getter)
                if not ok then return end
                return k, v
            end
    end

    local writeit
    writeit = function (t,oldindent,indent)
        local tp = type(t)
        if tp ~= 'string' and  tp ~= 'table' then
            putln(quote_if_necessary(tostring(t))..',')
        elseif tp == 'string' then
            -- if t:find('\n') then
            --     putln('[[\n'..t..']],')
            -- else
            --     putln(quote(t)..',')
            -- end
            --AAS
            putln(quote_string(t) ..",")
        elseif tp == 'table' then
            if tables[t] then
                putln('<cycle>,')
                return
            end
            tables[t] = true
            local newindent = indent..space
            putln('{')
            local used = {}
            if not not_clever then
                for i,val in ipairs(t) do
                    put(indent)
                    writeit(val,indent,newindent)
                    used[i] = true
                end
            end
            local ordered_keys = {}
            for k,v in pairs(t) do
               if type(k) ~= 'number' then
                  ordered_keys[#ordered_keys + 1] = k
               end
            end
            table.sort(ordered_keys, function (a, b)
                if type(a) == type(b) then
                    return tostring(a) < tostring(b)
                else
                    return type(a) < type(b)
                end
            end)
            local function write_entry (key, val)
                local tkey = type(key)
                local numkey = tkey == 'number'
                if not_clever then
                    key = tostring(key)
                    put(indent..index(numkey,key)..set)
                    writeit(val,indent,newindent)
                else
                    if not numkey or not used[key] then -- non-array indices
                        if tkey ~= 'string' then
                            key = tostring(key)
                        end
                        if numkey or not is_identifier(key) then
                            key = index(numkey,key)
                        end
                        put(indent..key..set)
                        writeit(val,indent,newindent)
                    end
                end
            end
            for i = 1, #ordered_keys do
                local key = ordered_keys[i]
                local val = t[key]
                write_entry(key, val)
            end
            for key,val in pairs(t) do
               if type(key) == 'number' then
                  write_entry(key, val)
               end
            end
            tables[t] = nil
            eat_last_comma()
            putln(oldindent..'},')
        else
            putln(tostring(t)..',')
        end
    end
    writeit(tbl,'',space)
    eat_last_comma()
    return concat(lines,#space > 0 and '\n' or '')
end

--- Dump a Lua table out to a file or stdout.
-- @tab t The table to write to a file or stdout.
-- @string[opt] filename File name to write too. Defaults to writing
-- to stdout.
function pretty.dump (t, filename)
    if not filename then
        print(pretty.write(t))
        return true
    else
        return utils.writefile(filename, pretty.write(t))
    end
end

--- Dump a series of arguments to stdout for debug purposes.
-- This function is attached to the module table `__call` method, to make it
-- extra easy to access. So the full:
--
--     print(require("pl.pretty").write({...}))
--
-- Can be shortened to:
--
--     require"pl.pretty" (...)
--
-- Any `nil` entries will be printed as `"<nil>"` to make them explicit.
-- @param ... the parameters to dump to stdout.
-- @usage
-- -- example debug output
-- require"pl.pretty" ("hello", nil, "world", { bye = "world", true} )
--
-- -- output:
-- {
--   ["arg 1"] = "hello",
--   ["arg 2"] = "<nil>",
--   ["arg 3"] = "world",
--   ["arg 4"] = {
--     true,
--     bye = "world"
--   }
-- }
function pretty.debug(...)
    local n = select("#", ...)
    local t = { ... }
    for i = 1, n do
        local value = t[i]
        if value == nil then
            value = "<nil>"
        end
        t[i] = nil
        t["arg " .. i] = value
    end

    print(pretty.write(t))
    return true
end


local memp,nump = {'B','KiB','MiB','GiB'},{'','K','M','B'}

local function comma (val)
    local thou = math.floor(val/1000)
    if thou > 0 then return comma(thou)..','.. tostring(val % 1000)
    else return tostring(val) end
end

--- Format large numbers nicely for human consumption.
-- @number num a number.
-- @string[opt] kind one of `'M'` (memory in `KiB`, `MiB`, etc.),
-- `'N'` (postfixes are `'K'`, `'M'` and `'B'`),
-- or `'T'` (use commas as thousands separator), `'N'` by default.
-- @int[opt] prec number of digits to use for `'M'` and `'N'`, `1` by default.
function pretty.number (num,kind,prec)
    local fmt = '%.'..(prec or 1)..'f%s'
    if kind == 'T' then
        return comma(num)
    else
        local postfixes, fact
        if kind == 'M' then
            fact = 1024
            postfixes = memp
        else
            fact = 1000
            postfixes = nump
        end
        local div = fact
        local k = 1
        while num >= div and k <= #postfixes do
            div = div * fact
            k = k + 1
        end
        div = div / fact
        if k > #postfixes then k = k - 1; div = div/fact end
        if k > 1 then
            return fmt:format(num/div,postfixes[k] or 'duh')
        else
            return num..postfixes[1]
        end
    end
end

return setmetatable(pretty, {
    __call = function(self, ...)
        return self.debug(...)
    end
})
