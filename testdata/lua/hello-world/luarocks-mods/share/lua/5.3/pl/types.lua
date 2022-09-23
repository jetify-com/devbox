---- Dealing with Detailed Type Information

-- Dependencies `pl.utils`
-- @module pl.types

local utils = require 'pl.utils'
local math_ceil = math.ceil
local assert_arg = utils.assert_arg
local types = {}

--- is the object either a function or a callable object?.
-- @param obj Object to check.
function types.is_callable (obj)
    return type(obj) == 'function' or getmetatable(obj) and getmetatable(obj).__call and true
end

--- is the object of the specified type?.
-- If the type is a string, then use type, otherwise compare with metatable.
--
-- NOTE: this function is imported from `utils.is_type`.
-- @param obj An object to check
-- @param tp The expected type
-- @function is_type
-- @see utils.is_type
types.is_type = utils.is_type

local fileMT = getmetatable(io.stdout)

--- a string representation of a type.
-- For tables and userdata with metatables, we assume that the metatable has a `_name`
-- field. If the field is not present it will return 'unknown table' or
-- 'unknown userdata'.
-- Lua file objects return the type 'file'.
-- @param obj an object
-- @return a string like 'number', 'table', 'file' or 'List'
function types.type (obj)
    local t = type(obj)
    if t == 'table' or t == 'userdata' then
        local mt = getmetatable(obj)
        if mt == fileMT then
            return 'file'
        elseif mt == nil then
            return t
        else
            -- TODO: the "unknown" is weird, it should just return the type
            return mt._name or "unknown "..t
        end
    else
        return t
    end
end

--- is this number an integer?
-- @param x a number
-- @raise error if x is not a number
-- @return boolean
function types.is_integer (x)
    return math_ceil(x)==x
end

--- Check if the object is "empty".
-- An object is considered empty if it is:
--
-- - `nil`
-- - a table without any items (key-value pairs or indexes)
-- - a string with no content ("")
-- - not a nil/table/string
-- @param o The object to check if it is empty.
-- @param ignore_spaces If the object is a string and this is true the string is
-- considered empty if it only contains spaces.
-- @return `true` if the object is empty, otherwise a falsy value.
function types.is_empty(o, ignore_spaces)
    if o == nil then
        return true
    elseif type(o) == "table" then
        return next(o) == nil
    elseif type(o) == "string" then
        return o == "" or (not not ignore_spaces and (not not o:find("^%s+$")))
    else
        return true
    end
end

local function check_meta (val)
    if type(val) == 'table' then return true end
    return getmetatable(val)
end

--- is an object 'array-like'?
-- An object is array like if:
--
-- - it is a table, or
-- - it has a metatable with `__len` and `__index` methods
--
-- NOTE: since `__len` is 5.2+, on 5.1 is usually returns `false` for userdata
-- @param val any value.
-- @return `true` if the object is array-like, otherwise a falsy value.
function types.is_indexable (val)
    local mt = check_meta(val)
    if mt == true then return true end
    return mt and mt.__len and mt.__index and true
end

--- can an object be iterated over with `pairs`?
-- An object is iterable if:
--
-- - it is a table, or
-- - it has a metatable with a `__pairs` meta method
--
-- NOTE: since `__pairs` is 5.2+, on 5.1 is usually returns `false` for userdata
-- @param val any value.
-- @return `true` if the object is iterable, otherwise a falsy value.
function types.is_iterable (val)
    local mt = check_meta(val)
    if mt == true then return true end
    return mt and mt.__pairs and true
end

--- can an object accept new key/pair values?
-- An object is iterable if:
--
-- - it is a table, or
-- - it has a metatable with a `__newindex` meta method
--
-- @param val any value.
-- @return `true` if the object is writeable, otherwise a falsy value.
function types.is_writeable (val)
    local mt = check_meta(val)
    if mt == true then return true end
    return mt and mt.__newindex and true
end

-- Strings that should evaluate to true.   -- TODO: add on/off ???
local trues = { yes=true, y=true, ["true"]=true, t=true, ["1"]=true }
-- Conditions types should evaluate to true.
local true_types = {
    boolean=function(o, true_strs, check_objs) return o end,
    string=function(o, true_strs, check_objs)
        o = o:lower()
        if trues[o] then
            return true
        end
        -- Check alternative user provided strings.
        for _,v in ipairs(true_strs or {}) do
            if type(v) == "string" and o == v:lower() then
                return true
            end
        end
        return false
    end,
    number=function(o, true_strs, check_objs) return o ~= 0 end,
    table=function(o, true_strs, check_objs) if check_objs and next(o) ~= nil then return true end return false end
}
--- Convert to a boolean value.
-- True values are:
--
-- * boolean: true.
-- * string: 'yes', 'y', 'true', 't', '1' or additional strings specified by `true_strs`.
-- * number: Any non-zero value.
-- * table: Is not empty and `check_objs` is true.
-- * everything else: Is not `nil` and `check_objs` is true.
--
-- @param o The object to evaluate.
-- @param[opt] true_strs optional Additional strings that when matched should evaluate to true. Comparison is case insensitive.
-- This should be a List of strings. E.g. "ja" to support German.
-- @param[opt] check_objs True if objects should be evaluated.
-- @return true if the input evaluates to true, otherwise false.
function types.to_bool(o, true_strs, check_objs)
    local true_func
    if true_strs then
        assert_arg(2, true_strs, "table")
    end
    true_func = true_types[type(o)]
    if true_func then
        return true_func(o, true_strs, check_objs)
    elseif check_objs and o ~= nil then
        return true
    end
    return false
end


return types
