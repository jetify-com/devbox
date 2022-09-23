--- Application support functions.
-- See @{01-introduction.md.Application_Support|the Guide}
--
-- Dependencies: `pl.utils`, `pl.path`
-- @module pl.app

local io,package,require = _G.io, _G.package, _G.require
local utils = require 'pl.utils'
local path = require 'pl.path'

local app = {}

--- return the name of the current script running.
-- The name will be the name as passed on the command line
-- @return string filename
function app.script_name()
    if _G.arg and _G.arg[0] then
        return _G.arg[0]
    end
    return utils.raise("No script name found")
end

--- prefixes the current script's path to the Lua module path.
-- Applies to both the source and the binary module paths. It makes it easy for
-- the main file of a multi-file program to access its modules in the same directory.
-- `base` allows these modules to be put in a specified subdirectory, to allow for
-- cleaner deployment and resolve potential conflicts between a script name and its
-- library directory.
--
-- Note: the path is prefixed, so it is searched first when requiring modules.
-- @string base optional base directory (absolute, or relative path).
-- @bool nofollow always use the invocation's directory, even if the invoked file is a symlink
-- @treturn string the current script's path with a trailing slash
function app.require_here (base, nofollow)
    local p = app.script_name()
    if not path.isabs(p) then
        p = path.join(path.currentdir(),p)
    end
    if not nofollow then
      local t = path.link_attrib(p)
      if t and t.mode == 'link' then
        t = t.target
        if not path.isabs(t) then
          t = path.join(path.dirname(p), t)
        end
        p = t
      end
    end
    p = path.normpath(path.dirname(p))
    if p:sub(-1,-1) ~= path.sep then
        p = p..path.sep
    end
    if base then
        if path.is_windows then
            base = base:gsub('/','\\')
        end
        if path.isabs(base) then
            p = base .. path.sep
        else
            p = p..base..path.sep
        end
    end
    local so_ext = path.is_windows and 'dll' or 'so'
    local lsep = package.path:find '^;' and '' or ';'
    local csep = package.cpath:find '^;' and '' or ';'
    package.path = ('%s?.lua;%s?%sinit.lua%s%s'):format(p,p,path.sep,lsep,package.path)
    package.cpath = ('%s?.%s%s%s'):format(p,so_ext,csep,package.cpath)
    return p
end

--- return a suitable path for files private to this application.
-- These will look like '~/.SNAME/file', with '~' as with expanduser and
-- SNAME is the name of the script without .lua extension.
-- If the directory does not exist, it will be created.
-- @string file a filename (w/out path)
-- @return a full pathname, or nil
-- @return cannot create directory error
-- @usage
-- -- when run from a script called 'testapp' (on Windows):
-- local app = require 'pl.app'
-- print(app.appfile 'test.txt')
-- -- C:\Documents and Settings\steve\.testapp\test.txt
function app.appfile(file)
    local sfullname, err = app.script_name()
    if not sfullname then return utils.raise(err) end
    local sname = path.basename(sfullname)
    local name = path.splitext(sname)
    local dir = path.join(path.expanduser('~'),'.'..name)
    if not path.isdir(dir) then
        local ret = path.mkdir(dir)
        if not ret then return utils.raise('cannot create '..dir) end
    end
    return path.join(dir,file)
end

--- return string indicating operating system.
-- @return 'Windows','OSX' or whatever uname returns (e.g. 'Linux')
function app.platform()
    if path.is_windows then
        return 'Windows'
    else
        local f = io.popen('uname')
        local res = f:read()
        if res == 'Darwin' then res = 'OSX' end
        f:close()
        return res
    end
end

--- return the full command-line used to invoke this script.
-- It will not include the scriptname itself, see `app.script_name`.
-- @return command-line
-- @return name of Lua program used
-- @usage
-- -- execute:  lua -lluacov -e 'print(_VERSION)' myscript.lua
--
-- -- myscript.lua
-- print(require("pl.app").lua())  --> "lua -lluacov -e 'print(_VERSION)'", "lua"
function app.lua()
    local args = _G.arg
    if not args then
        return utils.raise "not in a main program"
    end

    local cmd = {}
    local i = -1
    while true do
        table.insert(cmd, 1, args[i])
        if not args[i-1] then
            return utils.quote_arg(cmd), args[i]
        end
        i = i - 1
    end
end

--- parse command-line arguments into flags and parameters.
-- Understands GNU-style command-line flags; short (`-f`) and long (`--flag`).
--
-- These may be given a value with either '=' or ':' (`-k:2`,`--alpha=3.2`,`-n2`),
-- a number value can be given without a space. If the flag is marked
-- as having a value, then a space-separated value is also accepted (`-i hello`),
-- see the `flags_with_values` argument).
--
-- Multiple short args can be combined like so: ( `-abcd`).
--
-- When specifying the `flags_valid` parameter, its contents can also contain
-- aliasses, to convert short/long flags to the same output name. See the
-- example below.
--
-- Note: if a flag is repeated, the last value wins.
-- @tparam {string} args an array of strings (default is the global `arg`)
-- @tab flags_with_values any flags that take values, either list or hash
-- table e.g. `{ out=true }` or `{ "out" }`.
-- @tab flags_valid (optional) flags that are valid, either list or hashtable.
-- If not given, everything
-- will be accepted(everything in `flags_with_values` will automatically be allowed)
-- @return a table of flags (flag=value pairs)
-- @return an array of parameters
-- @raise if args is nil, then the global `args` must be available!
-- @usage
-- -- Simple form:
-- local flags, params = app.parse_args(nil,
--      { "hello", "world" },  -- list of flags taking values
--      { "l", "a", "b"})      -- list of allowed flags (value ones will be added)
--
-- -- More complex example using aliasses:
-- local valid = {
--     long = "l",           -- if 'l' is specified, it is reported as 'long'
--     new = { "n", "old" }, -- here both 'n' and 'old' will go into 'new'
-- }
-- local values = {
--     "value",   -- will automatically be added to the allowed set of flags
--     "new",     -- will mark 'n' and 'old' as requiring a value as well
-- }
-- local flags, params = app.parse_args(nil, values, valid)
--
-- -- command:  myapp.lua -l --old:hello --value world param1 param2
-- -- will yield:
-- flags = {
--     long = true,     -- input from 'l'
--     new = "hello",   -- input from 'old'
--     value = "world", -- allowed because it was in 'values', note: space separated!
-- }
-- params = {
--     [1] = "param1"
--     [2] = "param2"
-- }
function app.parse_args (args,flags_with_values, flags_valid)
    if not args then
        args = _G.arg
        if not args then utils.raise "Not in a main program: 'arg' not found" end
    end

    local with_values = {}
    for k,v in pairs(flags_with_values or {}) do
        if type(k) == "number" then
            k = v
        end
        with_values[k] = true
    end

    local valid
    if not flags_valid then
        -- if no allowed flags provided, we create a table that always returns
        -- the keyname, no matter what you look up
        valid = setmetatable({},{ __index = function(_, key) return key end })
    else
        valid = {}
        for k,aliasses in pairs(flags_valid) do
            if type(k) == "number" then         -- array/list entry
                k = aliasses
            end
            if type(aliasses) == "string" then  -- single alias
                aliasses = { aliasses }
            end
            if type(aliasses) == "table" then   -- list of aliasses
                -- it's the alternate name, so add the proper mappings
                for i, alias in ipairs(aliasses) do
                    valid[alias] = k
                end
            end
            valid[k] = k
        end
        do
            local new_with_values = {}  -- needed to prevent "invalid key to 'next'" error
            for k,v in pairs(with_values) do
                if not valid[k] then
                    valid[k] = k   -- add the with_value entry as a valid one
                    new_with_values[k] = true
                else
                    new_with_values[valid[k]] = true  --set, but by its alias
                end
            end
            with_values = new_with_values
        end
    end

    -- now check that all flags with values are reported as such under all
    -- of their aliasses
    for k, main_alias in pairs(valid) do
        if with_values[main_alias] then
            with_values[k] = true
        end
    end

    local _args = {}
    local flags = {}
    local i = 1
    while i <= #args do
        local a = args[i]
        local v = a:match('^-(.+)')
        local is_long
        if not v then
            -- we have a parameter
            _args[#_args+1] = a
        else
            -- it's a flag
            if v:find '^-' then
                is_long = true
                v = v:sub(2)
            end
            if with_values[v] then
                if i == #args or args[i+1]:find '^-' then
                    return utils.raise ("no value for '"..v.."'")
                end
                flags[valid[v]] = args[i+1]
                i = i + 1
            else
                -- a value can also be indicated with = or :
                local var,val =  utils.splitv (v,'[=:]', false, 2)
                var = var or v
                val = val or true
                if not is_long then
                    if #var > 1 then
                        if var:find '.%d+' then -- short flag, number value
                            val = var:sub(2)
                            var = var:sub(1,1)
                        else -- multiple short flags
                            for i = 1,#var do
                                local f = var:sub(i,i)
                                if not valid[f] then
                                    return utils.raise("unknown flag '"..f.."'")
                                else
                                    f = valid[f]
                                end
                                flags[f] = true
                            end
                            val = nil -- prevents use of var as a flag below
                        end
                    else  -- single short flag (can have value, defaults to true)
                        val = val or true
                    end
                end
                if val then
                    if not valid[var] then
                        return utils.raise("unknown flag '"..var.."'")
                    else
                        var = valid[var]
                    end
                    flags[var] = val
                end
            end
        end
        i = i + 1
    end
    return flags,_args
end

return app
