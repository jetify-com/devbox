--- Generally useful routines.
-- See  @{01-introduction.md.Generally_useful_functions|the Guide}.
--
-- Dependencies: `pl.compat`, all exported fields and functions from
-- `pl.compat` are also available in this module.
--
-- @module pl.utils
local format = string.format
local compat = require 'pl.compat'
local stdout = io.stdout
local append = table.insert
local concat = table.concat
local _unpack = table.unpack  -- always injected by 'compat'
local find = string.find
local sub = string.sub
local next = next
local floor = math.floor

local is_windows = compat.is_windows
local err_mode = 'default'
local raise
local operators
local _function_factories = {}


local utils = { _VERSION = "1.13.1" }
for k, v in pairs(compat) do utils[k] = v  end

--- Some standard patterns
-- @table patterns
utils.patterns = {
    FLOAT = '[%+%-%d]%d*%.?%d*[eE]?[%+%-]?%d*', -- floating point number
    INTEGER = '[+%-%d]%d*',                     -- integer number
    IDEN = '[%a_][%w_]*',                       -- identifier
    FILE = '[%a%.\\][:%][%w%._%-\\]*',          -- file
}


--- Standard meta-tables as used by other Penlight modules
-- @table stdmt
-- @field List the List metatable
-- @field Map the Map metatable
-- @field Set the Set metatable
-- @field MultiMap the MultiMap metatable
utils.stdmt = {
    List = {_name='List'},
    Map = {_name='Map'},
    Set = {_name='Set'},
    MultiMap = {_name='MultiMap'},
}


--- pack an argument list into a table.
-- @param ... any arguments
-- @return a table with field `n` set to the length
-- @function utils.pack
-- @see compat.pack
-- @see utils.npairs
-- @see utils.unpack
utils.pack = table.pack  -- added here to be symmetrical with unpack

--- unpack a table and return its contents.
--
-- NOTE: this implementation differs from the Lua implementation in the way
-- that this one DOES honor the `n` field in the table `t`, such that it is 'nil-safe'.
-- @param t table to unpack
-- @param[opt] i index from which to start unpacking, defaults to 1
-- @param[opt] j index of the last element to unpack, defaults to `t.n` or else `#t`
-- @return multiple return values from the table
-- @function utils.unpack
-- @see compat.unpack
-- @see utils.pack
-- @see utils.npairs
-- @usage
-- local t = table.pack(nil, nil, nil, 4)
-- local a, b, c, d = table.unpack(t)   -- this `unpack` is NOT nil-safe, so d == nil
--
-- local a, b, c, d = utils.unpack(t)   -- this is nil-safe, so d == 4
function utils.unpack(t, i, j)
    return _unpack(t, i or 1, j or t.n or #t)
end

--- print an arbitrary number of arguments using a format.
-- Output will be sent to `stdout`.
-- @param fmt The format (see `string.format`)
-- @param ... Extra arguments for format
function utils.printf(fmt, ...)
    utils.assert_string(1, fmt)
    utils.fprintf(stdout, fmt, ...)
end

--- write an arbitrary number of arguments to a file using a format.
-- @param f File handle to write to.
-- @param fmt The format (see `string.format`).
-- @param ... Extra arguments for format
function utils.fprintf(f,fmt,...)
    utils.assert_string(2,fmt)
    f:write(format(fmt,...))
end

do
    local function import_symbol(T,k,v,libname)
        local key = rawget(T,k)
        -- warn about collisions!
        if key and k ~= '_M' and k ~= '_NAME' and k ~= '_PACKAGE' and k ~= '_VERSION' then
            utils.fprintf(io.stderr,"warning: '%s.%s' will not override existing symbol\n",libname,k)
            return
        end
        rawset(T,k,v)
    end

    local function lookup_lib(T,t)
        for k,v in pairs(T) do
            if v == t then return k end
        end
        return '?'
    end

    local already_imported = {}

    --- take a table and 'inject' it into the local namespace.
    -- @param t The table (table), or module name (string), defaults to this `utils` module table
    -- @param T An optional destination table (defaults to callers environment)
    function utils.import(t,T)
        T = T or _G
        t = t or utils
        if type(t) == 'string' then
            t = require (t)
        end
        local libname = lookup_lib(T,t)
        if already_imported[t] then return end
        already_imported[t] = libname
        for k,v in pairs(t) do
            import_symbol(T,k,v,libname)
        end
    end
end

--- return either of two values, depending on a condition.
-- @param cond A condition
-- @param value1 Value returned if cond is truthy
-- @param value2 Value returned if cond is falsy
function utils.choose(cond, value1, value2)
    return cond and value1 or value2
end

--- convert an array of values to strings.
-- @param t a list-like table
-- @param[opt] temp (table) buffer to use, otherwise allocate
-- @param[opt] tostr custom tostring function, called with (value,index). Defaults to `tostring`.
-- @return the converted buffer
function utils.array_tostring (t,temp,tostr)
    temp, tostr = temp or {}, tostr or tostring
    for i = 1,#t do
        temp[i] = tostr(t[i],i)
    end
    return temp
end



--- is the object of the specified type?
-- If the type is a string, then use type, otherwise compare with metatable
-- @param obj An object to check
-- @param tp String of what type it should be
-- @return boolean
-- @usage utils.is_type("hello world", "string")   --> true
-- -- or check metatable
-- local my_mt = {}
-- local my_obj = setmetatable(my_obj, my_mt)
-- utils.is_type(my_obj, my_mt)  --> true
function utils.is_type (obj,tp)
    if type(tp) == 'string' then return type(obj) == tp end
    local mt = getmetatable(obj)
    return tp == mt
end



--- an iterator with indices, similar to `ipairs`, but with a range.
-- This is a nil-safe index based iterator that will return `nil` when there
-- is a hole in a list. To be safe ensure that table `t.n` contains the length.
-- @tparam table t the table to iterate over
-- @tparam[opt=1] integer i_start start index
-- @tparam[opt=t.n or #t] integer i_end end index
-- @tparam[opt=1] integer step step size
-- @treturn integer index
-- @treturn any value at index (which can be `nil`!)
-- @see utils.pack
-- @see utils.unpack
-- @usage
-- local t = utils.pack(nil, 123, nil)  -- adds an `n` field when packing
--
-- for i, v in utils.npairs(t, 2) do  -- start at index 2
--   t[i] = tostring(t[i])
-- end
--
-- -- t = { n = 3, [2] = "123", [3] = "nil" }
function utils.npairs(t, i_start, i_end, step)
  step = step or 1
  if step == 0 then
    error("iterator step-size cannot be 0", 2)
  end
  local i = (i_start or 1) - step
  i_end = i_end or t.n or #t
  if step < 0 then
    return function()
      i = i + step
      if i < i_end then
        return nil
      end
      return i, t[i]
    end

  else
    return function()
      i = i + step
      if i > i_end then
        return nil
      end
      return i, t[i]
    end
  end
end



--- an iterator over all non-integer keys (inverse of `ipairs`).
-- It will skip any key that is an integer number, so negative indices or an
-- array with holes will not return those either (so it returns slightly less than
-- 'the inverse of `ipairs`').
--
-- This uses `pairs` under the hood, so any value that is iterable using `pairs`
-- will work with this function.
-- @tparam table t the table to iterate over
-- @treturn key
-- @treturn value
-- @usage
-- local t = {
--   "hello",
--   "world",
--   hello = "hallo",
--   world = "Welt",
-- }
--
-- for k, v in utils.kpairs(t) do
--   print("German: ", v)
-- end
--
-- -- output;
-- -- German: hallo
-- -- German: Welt
function utils.kpairs(t)
  local index
  return function()
    local value
    while true do
      index, value = next(t, index)
      if type(index) ~= "number" or floor(index) ~= index then
        break
      end
    end
    return index, value
  end
end



--- Error handling
-- @section Error-handling

--- assert that the given argument is in fact of the correct type.
-- @param n argument index
-- @param val the value
-- @param tp the type
-- @param verify an optional verification function
-- @param msg an optional custom message
-- @param lev optional stack position for trace, default 2
-- @return the validated value
-- @raise if `val` is not the correct type
-- @usage
-- local param1 = assert_arg(1,"hello",'table')  --> error: argument 1 expected a 'table', got a 'string'
-- local param4 = assert_arg(4,'!@#$%^&*','string',path.isdir,'not a directory')
--      --> error: argument 4: '!@#$%^&*' not a directory
function utils.assert_arg (n,val,tp,verify,msg,lev)
    if type(val) ~= tp then
        error(("argument %d expected a '%s', got a '%s'"):format(n,tp,type(val)),lev or 2)
    end
    if verify and not verify(val) then
        error(("argument %d: '%s' %s"):format(n,val,msg),lev or 2)
    end
    return val
end

--- creates an Enum or constants lookup table for improved error handling.
-- This helps prevent magic strings in code by throwing errors for accessing
-- non-existing values, and/or converting strings/identifiers to other values.
--
-- Calling on the object does the same, but returns a soft error; `nil + err`, if
-- the call is succesful (the key exists), it will return the value.
--
-- When calling with varargs or an array the values will be equal to the keys.
-- The enum object is read-only.
-- @tparam table|vararg ... the input for the Enum. If varargs or an array then the
-- values in the Enum will be equal to the names (must be strings), if a hash-table
-- then values remain (any type), and the keys must be strings.
-- @return Enum object (read-only table/object)
-- @usage -- Enum access at runtime
-- local obj = {}
-- obj.MOVEMENT = utils.enum("FORWARD", "REVERSE", "LEFT", "RIGHT")
--
-- if current_movement == obj.MOVEMENT.FORWARD then
--   -- do something
--
-- elseif current_movement == obj.MOVEMENT.REVERES then
--   -- throws error due to typo 'REVERES', so a silent mistake becomes a hard error
--   -- "'REVERES' is not a valid value (expected one of: 'FORWARD', 'REVERSE', 'LEFT', 'RIGHT')"
--
-- end
-- @usage -- standardized error codes
-- local obj = {
--   ERR = utils.enum {
--     NOT_FOUND = "the item was not found",
--     OUT_OF_BOUNDS = "the index is outside the allowed range"
--   },
--
--   some_method = function(self)
--     return self.ERR.OUT_OF_BOUNDS
--   end,
-- }
--
-- local result, err = obj:some_method()
-- if not result then
--   if err == obj.ERR.NOT_FOUND then
--     -- check on error code, not magic strings
--
--   else
--     -- return the error description, contained in the constant
--     return nil, "error: "..err  -- "error: the index is outside the allowed range"
--   end
-- end
-- @usage -- validating/converting user-input
-- local color = "purple"
-- local ansi_colors = utils.enum {
--   black     = 30,
--   red       = 31,
--   green     = 32,
-- }
-- local color_code, err = ansi_colors(color) -- calling on the object, returns the value from the enum
-- if not color_code then
--   print("bad 'color', " .. err)
--   -- "bad 'color', 'purple' is not a valid value (expected one of: 'black', 'red', 'green')"
--   os.exit(1)
-- end
function utils.enum(...)
  local first = select(1, ...)
  local enum = {}
  local lst

  if type(first) ~= "table" then
    -- vararg with strings
    lst = utils.pack(...)
    for i, value in utils.npairs(lst) do
      utils.assert_arg(i, value, "string")
      enum[value] = value
    end

  else
    -- table/array with values
    utils.assert_arg(1, first, "table")
    lst = {}
    -- first add array part
    for i, value in ipairs(first) do
      if type(value) ~= "string" then
        error(("expected 'string' but got '%s' at index %d"):format(type(value), i), 2)
      end
      lst[i] = value
      enum[value] = value
    end
    -- add key-ed part
    for key, value in utils.kpairs(first) do
      if type(key) ~= "string" then
        error(("expected key to be 'string' but got '%s'"):format(type(key)), 2)
      end
      if enum[key] then
        error(("duplicate entry in array and hash part: '%s'"):format(key), 2)
      end
      enum[key] = value
      lst[#lst+1] = key
    end
  end

  if not lst[1] then
    error("expected at least 1 entry", 2)
  end

  local valid = "(expected one of: '" .. concat(lst, "', '") .. "')"
  setmetatable(enum, {
    __index = function(self, key)
      error(("'%s' is not a valid value %s"):format(tostring(key), valid), 2)
    end,
    __newindex = function(self, key, value)
      error("the Enum object is read-only", 2)
    end,
    __call = function(self, key)
      if type(key) == "string" then
        local v = rawget(self, key)
        if v ~= nil then
          return v
        end
      end
      return nil, ("'%s' is not a valid value %s"):format(tostring(key), valid)
    end
  })

  return enum
end


--- process a function argument.
-- This is used throughout Penlight and defines what is meant by a function:
-- Something that is callable, or an operator string as defined by <code>pl.operator</code>,
-- such as '>' or '#'. If a function factory has been registered for the type, it will
-- be called to get the function.
-- @param idx argument index
-- @param f a function, operator string, or callable object
-- @param msg optional error message
-- @return a callable
-- @raise if idx is not a number or if f is not callable
function utils.function_arg (idx,f,msg)
    utils.assert_arg(1,idx,'number')
    local tp = type(f)
    if tp == 'function' then return f end  -- no worries!
    -- ok, a string can correspond to an operator (like '==')
    if tp == 'string' then
        if not operators then operators = require 'pl.operator'.optable end
        local fn = operators[f]
        if fn then return fn end
        local fn, err = utils.string_lambda(f)
        if not fn then error(err..': '..f) end
        return fn
    elseif tp == 'table' or tp == 'userdata' then
        local mt = getmetatable(f)
        if not mt then error('not a callable object',2) end
        local ff = _function_factories[mt]
        if not ff then
            if not mt.__call then error('not a callable object',2) end
            return f
        else
            return ff(f) -- we have a function factory for this type!
        end
    end
    if not msg then msg = " must be callable" end
    if idx > 0 then
        error("argument "..idx..": "..msg,2)
    else
        error(msg,2)
    end
end


--- assert the common case that the argument is a string.
-- @param n argument index
-- @param val a value that must be a string
-- @return the validated value
-- @raise val must be a string
-- @usage
-- local val = 42
-- local param2 = utils.assert_string(2, val) --> error: argument 2 expected a 'string', got a 'number'
function utils.assert_string (n, val)
    return utils.assert_arg(n,val,'string',nil,nil,3)
end

--- control the error strategy used by Penlight.
-- This is a global setting that controls how `utils.raise` behaves:
--
-- - 'default': return `nil + error` (this is the default)
-- - 'error': throw a Lua error
-- - 'quit': exit the program
--
-- @param mode either 'default', 'quit'  or 'error'
-- @see utils.raise
function utils.on_error (mode)
    mode = tostring(mode)
    if ({['default'] = 1, ['quit'] = 2, ['error'] = 3})[mode] then
      err_mode = mode
    else
      -- fail loudly
      local err = "Bad argument expected string; 'default', 'quit', or 'error'. Got '"..tostring(mode).."'"
      if err_mode == 'default' then
        error(err, 2)  -- even in 'default' mode fail loud in this case
      end
      raise(err)
    end
end

--- used by Penlight functions to return errors. Its global behaviour is controlled
-- by `utils.on_error`.
-- To use this function you MUST use it in conjunction with `return`, since it might
-- return `nil + error`.
-- @param err the error string.
-- @see utils.on_error
-- @usage
-- if some_condition then
--   return utils.raise("some condition was not met")  -- MUST use 'return'!
-- end
function utils.raise (err)
    if err_mode == 'default' then
        return nil, err
    elseif err_mode == 'quit' then
        return utils.quit(err)
    else
        error(err, 2)
    end
end
raise = utils.raise



--- File handling
-- @section files

--- return the contents of a file as a string
-- @param filename The file path
-- @param is_bin open in binary mode
-- @return file contents
function utils.readfile(filename,is_bin)
    local mode = is_bin and 'b' or ''
    utils.assert_string(1,filename)
    local f,open_err = io.open(filename,'r'..mode)
    if not f then return raise (open_err) end
    local res,read_err = f:read('*a')
    f:close()
    if not res then
        -- Errors in io.open have "filename: " prefix,
        -- error in file:read don't, add it.
        return raise (filename..": "..read_err)
    end
    return res
end

--- write a string to a file
-- @param filename The file path
-- @param str The string
-- @param is_bin open in binary mode
-- @return true or nil
-- @return error message
-- @raise error if filename or str aren't strings
function utils.writefile(filename,str,is_bin)
    local mode = is_bin and 'b' or ''
    utils.assert_string(1,filename)
    utils.assert_string(2,str)
    local f,err = io.open(filename,'w'..mode)
    if not f then return raise(err) end
    local ok, write_err = f:write(str)
    f:close()
    if not ok then
        -- Errors in io.open have "filename: " prefix,
        -- error in file:write don't, add it.
        return raise (filename..": "..write_err)
    end
    return true
end

--- return the contents of a file as a list of lines
-- @param filename The file path
-- @return file contents as a table
-- @raise error if filename is not a string
function utils.readlines(filename)
    utils.assert_string(1,filename)
    local f,err = io.open(filename,'r')
    if not f then return raise(err) end
    local res = {}
    for line in f:lines() do
        append(res,line)
    end
    f:close()
    return res
end

--- OS functions
-- @section OS-functions

--- execute a shell command and return the output.
-- This function redirects the output to tempfiles and returns the content of those files.
-- @param cmd a shell command
-- @param bin boolean, if true, read output as binary file
-- @return true if successful
-- @return actual return code
-- @return stdout output (string)
-- @return errout output (string)
function utils.executeex(cmd, bin)
    local outfile = os.tmpname()
    local errfile = os.tmpname()

    if is_windows and not outfile:find(':') then
        outfile = os.getenv('TEMP')..outfile
        errfile = os.getenv('TEMP')..errfile
    end
    cmd = cmd .. " > " .. utils.quote_arg(outfile) .. " 2> " .. utils.quote_arg(errfile)

    local success, retcode = utils.execute(cmd)
    local outcontent = utils.readfile(outfile, bin)
    local errcontent = utils.readfile(errfile, bin)
    os.remove(outfile)
    os.remove(errfile)
    return success, retcode, (outcontent or ""), (errcontent or "")
end

--- Quote and escape an argument of a command.
-- Quotes a single (or list of) argument(s) of a command to be passed
-- to `os.execute`, `pl.utils.execute` or `pl.utils.executeex`.
-- @param argument (string or table/list) the argument to quote. If a list then
-- all arguments in the list will be returned as a single string quoted.
-- @return quoted and escaped argument.
-- @usage
-- local options = utils.quote_arg {
--     "-lluacov",
--     "-e",
--     "utils = print(require('pl.utils')._VERSION",
-- }
-- -- returns: -lluacov -e 'utils = print(require('\''pl.utils'\'')._VERSION'
function utils.quote_arg(argument)
    if type(argument) == "table" then
        -- encode an entire table
        local r = {}
        for i, arg in ipairs(argument) do
            r[i] = utils.quote_arg(arg)
        end

        return concat(r, " ")
    end
    -- only a single argument
    if is_windows then
        if argument == "" or argument:find('[ \f\t\v]') then
            -- Need to quote the argument.
            -- Quotes need to be escaped with backslashes;
            -- additionally, backslashes before a quote, escaped or not,
            -- need to be doubled.
            -- See documentation for CommandLineToArgvW Windows function.
            argument = '"' .. argument:gsub([[(\*)"]], [[%1%1\"]]):gsub([[\+$]], "%0%0") .. '"'
        end

        -- os.execute() uses system() C function, which on Windows passes command
        -- to cmd.exe. Escape its special characters.
        return (argument:gsub('["^<>!|&%%]', "^%0"))
    else
        if argument == "" or argument:find('[^a-zA-Z0-9_@%+=:,./-]') then
            -- To quote arguments on posix-like systems use single quotes.
            -- To represent an embedded single quote close quoted string ('),
            -- add escaped quote (\'), open quoted string again (').
            argument = "'" .. argument:gsub("'", [['\'']]) .. "'"
        end

        return argument
    end
end

--- error out of this program gracefully.
-- @param[opt] code The exit code, defaults to -`1` if omitted
-- @param msg The exit message will be sent to `stderr` (will be formatted with the extra parameters)
-- @param ... extra arguments for message's format'
-- @see utils.fprintf
-- @usage utils.quit(-1, "Error '%s' happened", "42")
-- -- is equivalent to
-- utils.quit("Error '%s' happened", "42")  --> Error '42' happened
function utils.quit(code, msg, ...)
    if type(code) == 'string' then
        utils.fprintf(io.stderr, code, msg, ...)
        io.stderr:write('\n')
        code = -1 -- TODO: this is odd, see the test. Which returns 255 as exit code
    elseif msg then
        utils.fprintf(io.stderr, msg, ...)
        io.stderr:write('\n')
    end
    os.exit(code, true)
end


--- String functions
-- @section string-functions

--- escape any Lua 'magic' characters in a string
-- @param s The input string
function utils.escape(s)
    utils.assert_string(1,s)
    return (s:gsub('[%-%.%+%[%]%(%)%$%^%%%?%*]','%%%1'))
end

--- split a string into a list of strings separated by a delimiter.
-- @param s The input string
-- @param re optional A Lua string pattern; defaults to '%s+'
-- @param plain optional If truthy don't use Lua patterns
-- @param n optional maximum number of elements (if there are more, the last will remian un-split)
-- @return a list-like table
-- @raise error if s is not a string
-- @see splitv
function utils.split(s,re,plain,n)
    utils.assert_string(1,s)
    local i1,ls = 1,{}
    if not re then re = '%s+' end
    if re == '' then return {s} end
    while true do
        local i2,i3 = find(s,re,i1,plain)
        if not i2 then
            local last = sub(s,i1)
            if last ~= '' then append(ls,last) end
            if #ls == 1 and ls[1] == '' then
                return {}
            else
                return ls
            end
        end
        append(ls,sub(s,i1,i2-1))
        if n and #ls == n then
            ls[#ls] = sub(s,i1)
            return ls
        end
        i1 = i3+1
    end
end

--- split a string into a number of return values.
-- Identical to `split` but returns multiple sub-strings instead of
-- a single list of sub-strings.
-- @param s the string
-- @param re A Lua string pattern; defaults to '%s+'
-- @param plain don't use Lua patterns
-- @param n optional maximum number of splits
-- @return n values
-- @usage first,next = splitv('user=jane=doe','=', false, 2)
-- assert(first == "user")
-- assert(next == "jane=doe")
-- @see split
function utils.splitv (s,re, plain, n)
    return _unpack(utils.split(s,re, plain, n))
end


--- Functional
-- @section functional


--- 'memoize' a function (cache returned value for next call).
-- This is useful if you have a function which is relatively expensive,
-- but you don't know in advance what values will be required, so
-- building a table upfront is wasteful/impossible.
-- @param func a function of at least one argument
-- @return a function with at least one argument, which is used as the key.
function utils.memoize(func)
    local cache = {}
    return function(k)
        local res = cache[k]
        if res == nil then
            res = func(k)
            cache[k] = res
        end
        return res
    end
end


--- associate a function factory with a type.
-- A function factory takes an object of the given type and
-- returns a function for evaluating it
-- @tab mt metatable
-- @func fun a callable that returns a function
function utils.add_function_factory (mt,fun)
    _function_factories[mt] = fun
end

local function _string_lambda(f)
    if f:find '^|' or f:find '_' then
        local args,body = f:match '|([^|]*)|(.+)'
        if f:find '_' then
            args = '_'
            body = f
        else
            if not args then return raise 'bad string lambda' end
        end
        local fstr = 'return function('..args..') return '..body..' end'
        local fn,err = utils.load(fstr)
        if not fn then return raise(err) end
        fn = fn()
        return fn
    else
        return raise 'not a string lambda'
    end
end


--- an anonymous function as a string. This string is either of the form
-- '|args| expression' or is a function of one argument, '_'
-- @param lf function as a string
-- @return a function
-- @function utils.string_lambda
-- @usage
-- string_lambda '|x|x+1' (2) == 3
-- string_lambda '_+1' (2) == 3
utils.string_lambda = utils.memoize(_string_lambda)


--- bind the first argument of the function to a value.
-- @param fn a function of at least two values (may be an operator string)
-- @param p a value
-- @return a function such that f(x) is fn(p,x)
-- @raise same as @{function_arg}
-- @see func.bind1
-- @usage local function f(msg, name)
--   print(msg .. " " .. name)
-- end
--
-- local hello = utils.bind1(f, "Hello")
--
-- print(hello("world"))     --> "Hello world"
-- print(hello("sunshine"))  --> "Hello sunshine"
function utils.bind1 (fn,p)
    fn = utils.function_arg(1,fn)
    return function(...) return fn(p,...) end
end


--- bind the second argument of the function to a value.
-- @param fn a function of at least two values (may be an operator string)
-- @param p a value
-- @return a function such that f(x) is fn(x,p)
-- @raise same as @{function_arg}
-- @usage local function f(a, b, c)
--   print(a .. " " .. b .. " " .. c)
-- end
--
-- local hello = utils.bind1(f, "world")
--
-- print(hello("Hello", "!"))  --> "Hello world !"
-- print(hello("Bye", "?"))    --> "Bye world ?"
function utils.bind2 (fn,p)
    fn = utils.function_arg(1,fn)
    return function(x,...) return fn(x,p,...) end
end




--- Deprecation
-- @section deprecation

do
  -- the default implementation
  local deprecation_func = function(msg, trace)
    if trace then
      warn(msg, "\n", trace)  -- luacheck: ignore
    else
      warn(msg)  -- luacheck: ignore
    end
  end

  --- Sets a deprecation warning function.
  -- An application can override this function to support proper output of
  -- deprecation warnings. The warnings can be generated from libraries or
  -- functions by calling `utils.raise_deprecation`. The default function
  -- will write to the 'warn' system (introduced in Lua 5.4, or the compatibility
  -- function from the `compat` module for earlier versions).
  --
  -- Note: only applications should set/change this function, libraries should not.
  -- @param func a callback with signature: `function(msg, trace)` both arguments are strings, the latter being optional.
  -- @see utils.raise_deprecation
  -- @usage
  -- -- write to the Nginx logs with OpenResty
  -- utils.set_deprecation_func(function(msg, trace)
  --   ngx.log(ngx.WARN, msg, (trace and (" " .. trace) or nil))
  -- end)
  --
  -- -- disable deprecation warnings
  -- utils.set_deprecation_func()
  function utils.set_deprecation_func(func)
    if func == nil then
      deprecation_func = function() end
    else
      utils.assert_arg(1, func, "function")
      deprecation_func = func
    end
  end

  --- raises a deprecation warning.
  -- For options see the usage example below.
  --
  -- Note: the `opts.deprecated_after` field is the last version in which
  -- a feature or option was NOT YET deprecated! Because when writing the code it
  -- is quite often not known in what version the code will land. But the last
  -- released version is usually known.
  -- @param opts options table
  -- @see utils.set_deprecation_func
  -- @usage
  -- warn("@on")   -- enable Lua warnings, they are usually off by default
  --
  -- function stringx.islower(str)
  --   raise_deprecation {
  --     source = "Penlight " .. utils._VERSION,                   -- optional
  --     message = "function 'islower' was renamed to 'is_lower'", -- required
  --     version_removed = "2.0.0",                                -- optional
  --     deprecated_after = "1.2.3",                               -- optional
  --     no_trace = true,                                          -- optional
  --   }
  --   return stringx.is_lower(str)
  -- end
  -- -- output: "[Penlight 1.9.2] function 'islower' was renamed to 'is_lower' (deprecated after 1.2.3, scheduled for removal in 2.0.0)"
  function utils.raise_deprecation(opts)
    utils.assert_arg(1, opts, "table")
    if type(opts.message) ~= "string" then
      error("field 'message' of the options table must be a string", 2)
    end
    local trace
    if not opts.no_trace then
      trace = debug.traceback("", 2):match("[\n%s]*(.-)$")
    end
    local msg
    if opts.deprecated_after and opts.version_removed then
      msg = (" (deprecated after %s, scheduled for removal in %s)"):format(
        tostring(opts.deprecated_after), tostring(opts.version_removed))
    elseif opts.deprecated_after then
      msg = (" (deprecated after %s)"):format(tostring(opts.deprecated_after))
    elseif opts.version_removed then
      msg = (" (scheduled for removal in %s)"):format(tostring(opts.version_removed))
    else
      msg = ""
    end

    msg = opts.message .. msg

    if opts.source then
      msg = "[" .. opts.source .."] " .. msg
    else
      if msg:sub(1,1) == "@" then
        -- in Lua 5.4 "@" prefixed messages are control messages to the warn system
        error("message cannot start with '@'", 2)
      end
    end

    deprecation_func(msg, trace)
  end

end


return utils


