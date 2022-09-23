----------------
--- Lua 5.1/5.2/5.3 compatibility.
-- Injects `table.pack`, `table.unpack`, and `package.searchpath` in the global
-- environment, to make sure they are available for Lua 5.1 and LuaJIT.
--
-- All other functions are exported as usual in the returned module table.
--
-- NOTE: everything in this module is also available in `pl.utils`.
-- @module pl.compat
local compat = {}

--- boolean flag this is Lua 5.1 (or LuaJIT).
-- @field lua51
compat.lua51 = _VERSION == 'Lua 5.1'

--- boolean flag this is LuaJIT.
-- @field jit
compat.jit = (tostring(assert):match('builtin') ~= nil)

--- boolean flag this is LuaJIT with 5.2 compatibility compiled in.
-- @field jit52
if compat.jit then
    -- 'goto' is a keyword when 52 compatibility is enabled in LuaJit
    compat.jit52 = not loadstring("local goto = 1")
end

--- the directory separator character for the current platform.
-- @field dir_separator
compat.dir_separator = _G.package.config:sub(1,1)

--- boolean flag this is a Windows platform.
-- @field is_windows
compat.is_windows = compat.dir_separator == '\\'

--- execute a shell command, in a compatible and platform independent way.
-- This is a compatibility function that returns the same for Lua 5.1 and
-- Lua 5.2+.
--
-- NOTE: Windows systems can use signed 32bit integer exitcodes. Posix systems
-- only use exitcodes 0-255, anything else is undefined.
--
-- NOTE2: In Lua 5.2 and 5.3 a Windows exitcode of -1 would not properly be
-- returned, this function will return it properly for all versions.
-- @param cmd a shell command
-- @return true if successful
-- @return actual return code
function compat.execute(cmd)
    local res1,res2,res3 = os.execute(cmd)
    if res2 == "No error" and res3 == 0 and compat.is_windows then
      -- os.execute bug in Lua 5.2/5.3 not reporting -1 properly on Windows
      -- this was fixed in 5.4
      res3 = -1
    end
    if compat.lua51 and not compat.jit52 then
        if compat.is_windows then
            return res1==0,res1
        else
            res1 = res1 > 255 and res1 / 256 or res1
            return res1==0,res1
        end
    else
        if compat.is_windows then
            return res3==0,res3
        else
            return not not res1,res3
        end
    end
end

----------------
-- Load Lua code as a text or binary chunk (in a Lua 5.2 compatible way).
-- @param ld code string or loader
-- @param[opt] source name of chunk for errors
-- @param[opt] mode 'b', 't' or 'bt'
-- @param[opt] env environment to load the chunk in
-- @function compat.load

---------------
-- Get environment of a function (in a Lua 5.1 compatible way).
-- Not 100% compatible, so with Lua 5.2 it may return nil for a function with no
-- global references!
-- Based on code by [Sergey Rozhenko](http://lua-users.org/lists/lua-l/2010-06/msg00313.html)
-- @param f a function or a call stack reference
-- @function compat.getfenv

---------------
-- Set environment of a function (in a Lua 5.1 compatible way).
-- @param f a function or a call stack reference
-- @param env a table that becomes the new environment of `f`
-- @function compat.setfenv

if compat.lua51 then -- define Lua 5.2 style load()
    if not compat.jit then -- but LuaJIT's load _is_ compatible
        local lua51_load = load
        function compat.load(str,src,mode,env)
            local chunk,err
            if type(str) == 'string' then
                if str:byte(1) == 27 and not (mode or 'bt'):find 'b' then
                    return nil,"attempt to load a binary chunk"
                end
                chunk,err = loadstring(str,src)
            else
                chunk,err = lua51_load(str,src)
            end
            if chunk and env then setfenv(chunk,env) end
            return chunk,err
        end
    else
        compat.load = load
    end
    compat.setfenv, compat.getfenv = setfenv, getfenv
else
    compat.load = load
    -- setfenv/getfenv replacements for Lua 5.2
    -- by Sergey Rozhenko
    -- http://lua-users.org/lists/lua-l/2010-06/msg00313.html
    -- Roberto Ierusalimschy notes that it is possible for getfenv to return nil
    -- in the case of a function with no globals:
    -- http://lua-users.org/lists/lua-l/2010-06/msg00315.html
    function compat.setfenv(f, t)
        f = (type(f) == 'function' and f or debug.getinfo(f + 1, 'f').func)
        local name
        local up = 0
        repeat
            up = up + 1
            name = debug.getupvalue(f, up)
        until name == '_ENV' or name == nil
        if name then
            debug.upvaluejoin(f, up, function() return name end, 1) -- use unique upvalue
            debug.setupvalue(f, up, t)
        end
        if f ~= 0 then return f end
    end

    function compat.getfenv(f)
        local f = f or 0
        f = (type(f) == 'function' and f or debug.getinfo(f + 1, 'f').func)
        local name, val
        local up = 0
        repeat
            up = up + 1
            name, val = debug.getupvalue(f, up)
        until name == '_ENV' or name == nil
        return val
    end
end


--- Global exported functions (for Lua 5.1 & LuaJIT)
-- @section lua52

--- pack an argument list into a table.
-- @param ... any arguments
-- @return a table with field n set to the length
-- @function table.pack
if not table.pack then
    function table.pack (...)       -- luacheck: ignore
        return {n=select('#',...); ...}
    end
end

--- unpack a table and return the elements.
--
-- NOTE: this version does NOT honor the n field, and hence it is not nil-safe.
-- See `utils.unpack` for a version that is nil-safe.
-- @param t table to unpack
-- @param[opt] i index from which to start unpacking, defaults to 1
-- @param[opt] j index of the last element to unpack, defaults to #t
-- @return multiple return values from the table
-- @function table.unpack
-- @see utils.unpack
if not table.unpack then
    table.unpack = unpack           -- luacheck: ignore
end

--- return the full path where a file name would be matched.
-- This function was introduced in Lua 5.2, so this compatibility version
-- will be injected in Lua 5.1 engines.
-- @string name file name, possibly dotted
-- @string path a path-template in the same form as package.path or package.cpath
-- @string[opt] sep template separate character to be replaced by path separator. Default: "."
-- @string[opt] rep the path separator to use, defaults to system separator. Default; "/" on Unixes, "\" on Windows.
-- @see path.package_path
-- @function package.searchpath
-- @return on success: path of the file
-- @return on failure: nil, error string listing paths tried
if not package.searchpath then
    function package.searchpath (name,path,sep,rep)    -- luacheck: ignore
        if type(name) ~= "string" then
            error(("bad argument #1 to 'searchpath' (string expected, got %s)"):format(type(path)), 2)
        end
        if type(path) ~= "string" then
            error(("bad argument #2 to 'searchpath' (string expected, got %s)"):format(type(path)), 2)
        end
        if sep ~= nil and type(sep) ~= "string" then
            error(("bad argument #3 to 'searchpath' (string expected, got %s)"):format(type(path)), 2)
        end
        if rep ~= nil and type(rep) ~= "string" then
            error(("bad argument #4 to 'searchpath' (string expected, got %s)"):format(type(path)), 2)
        end
        sep = sep or "."
        rep = rep or compat.dir_separator
        do
          local s, e = name:find(sep, nil, true)
          while s do
            name = name:sub(1, s-1) .. rep .. name:sub(e+1, -1)
            s, e = name:find(sep, s + #rep + 1, true)
          end
        end
        local tried = {}
        for m in path:gmatch('[^;]+') do
            local nm = m:gsub('?', name)
            tried[#tried+1] = nm
            local f = io.open(nm,'r')
            if f then f:close(); return nm end
        end
        return nil, "\tno file '" .. table.concat(tried, "'\n\tno file '") .. "'"
    end
end

--- Global exported functions (for Lua < 5.4)
-- @section lua54

--- raise a warning message.
-- This functions mimics the `warn` function added in Lua 5.4.
-- @function warn
-- @param ... any arguments
if not rawget(_G, "warn") then
    local enabled = false
    local function warn(arg1, ...)
        if type(arg1) == "string" and arg1:sub(1, 1) == "@" then
            -- control message
            if arg1 == "@on" then
                enabled = true
                return
            end
            if arg1 == "@off" then
                enabled = false
                return
            end
            return -- ignore unknown control messages
        end
        if enabled then
          io.stderr:write("Lua warning: ", arg1, ...)
          io.stderr:write("\n")
        end
    end
    -- use rawset to bypass OpenResty's protection of global scope
    rawset(_G, "warn", warn)
end

return compat
