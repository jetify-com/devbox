--- Provides a reuseable and convenient framework for creating classes in Lua.
-- Two possible notations:
--
--    B = class(A)
--    class.B(A)
--
-- The latter form creates a named class within the current environment. Note
-- that this implicitly brings in `pl.utils` as a dependency.
--
-- See the Guide for further @{01-introduction.md.Simplifying_Object_Oriented_Programming_in_Lua|discussion}
-- @module pl.class

local error, getmetatable, io, pairs, rawget, rawset, setmetatable, tostring, type =
    _G.error, _G.getmetatable, _G.io, _G.pairs, _G.rawget, _G.rawset, _G.setmetatable, _G.tostring, _G.type
local compat

-- this trickery is necessary to prevent the inheritance of 'super' and
-- the resulting recursive call problems.
local function call_ctor (c,obj,...)
    local init = rawget(c,'_init')
    local parent_with_init = rawget(c,'_parent_with_init')

    if parent_with_init then
        if not init then -- inheriting an init
            init = rawget(parent_with_init, '_init')
            parent_with_init = rawget(parent_with_init, '_parent_with_init')
        end
        if parent_with_init then -- super() points to one above whereever _init came from
            rawset(obj,'super',function(obj,...)
                call_ctor(parent_with_init,obj,...)
            end)
        end
    else
        -- Without this, calling super() where none exists will sometimes loop and stack overflow
        rawset(obj,'super',nil)
    end

    local res = init(obj,...)
    if parent_with_init then -- If this execution of call_ctor set a super, unset it
        rawset(obj,'super',nil)
    end

    return res
end

--- initializes an __instance__ upon creation.
-- @function class:_init
-- @param ... parameters passed to the constructor
-- @usage local Cat = class()
-- function Cat:_init(name)
--   --self:super(name)   -- call the ancestor initializer if needed
--   self.name = name
-- end
--
-- local pussycat = Cat("pussycat")
-- print(pussycat.name)  --> pussycat

--- checks whether an __instance__ is derived from some class.
-- Works the other way around as `class_of`. It has two ways of using;
-- 1) call with a class to check against, 2) call without params.
-- @function instance:is_a
-- @param some_class class to check against, or `nil` to return the class
-- @return `true` if `instance` is derived from `some_class`, or if `some_class == nil` then
-- it returns the class table of the instance
-- @usage local pussycat = Lion()  -- assuming Lion derives from Cat
-- if pussycat:is_a(Cat) then
--   -- it's true, it is a Lion, but also a Cat
-- end
--
-- if pussycat:is_a() == Lion then
--   -- It's true
-- end
local function is_a(self,klass)
    if klass == nil then
        -- no class provided, so return the class this instance is derived from
        return getmetatable(self)
    end
    local m = getmetatable(self)
    if not m then return false end --*can't be an object!
    while m do
        if m == klass then return true end
        m = rawget(m,'_base')
    end
    return false
end

--- checks whether an __instance__ is derived from some class.
-- Works the other way around as `is_a`.
-- @function some_class:class_of
-- @param some_instance instance to check against
-- @return `true` if `some_instance` is derived from `some_class`
-- @usage local pussycat = Lion()  -- assuming Lion derives from Cat
-- if Cat:class_of(pussycat) then
--   -- it's true
-- end
local function class_of(klass,obj)
    if type(klass) ~= 'table' or not rawget(klass,'is_a') then return false end
    return klass.is_a(obj,klass)
end

--- cast an object to another class.
-- It is not clever (or safe!) so use carefully.
-- @param some_instance the object to be changed
-- @function some_class:cast
local function cast (klass, obj)
    return setmetatable(obj,klass)
end


local function _class_tostring (obj)
    local mt = obj._class
    local name = rawget(mt,'_name')
    setmetatable(obj,nil)
    local str = tostring(obj)
    setmetatable(obj,mt)
    if name then str = name ..str:gsub('table','') end
    return str
end

local function tupdate(td,ts,dont_override)
    for k,v in pairs(ts) do
        if not dont_override or td[k] == nil then
            td[k] = v
        end
    end
end

local function _class(base,c_arg,c)
    -- the class `c` will be the metatable for all its objects,
    -- and they will look up their methods in it.
    local mt = {}   -- a metatable for the class to support __call and _handler
    -- can define class by passing it a plain table of methods
    local plain = type(base) == 'table' and not getmetatable(base)
    if plain then
        c = base
        base = c._base
    else
        c = c or {}
    end

    if type(base) == 'table' then
        -- our new class is a shallow copy of the base class!
        -- but be careful not to wipe out any methods we have been given at this point!
        tupdate(c,base,plain)
        c._base = base
        -- inherit the 'not found' handler, if present
        if rawget(c,'_handler') then mt.__index = c._handler end
    elseif base ~= nil then
        error("must derive from a table type",3)
    end

    c.__index = c
    setmetatable(c,mt)
    if not plain then
        if base and rawget(base,'_init') then c._parent_with_init = base end -- For super and inherited init
        c._init = nil
    end

    if base and rawget(base,'_class_init') then
        base._class_init(c,c_arg)
    end

    -- expose a ctor which can be called by <classname>(<args>)
    mt.__call = function(class_tbl,...)
        local obj
        if rawget(c,'_create') then obj = c._create(...) end
        if not obj then obj = {} end
        setmetatable(obj,c)

        if rawget(c,'_init') or rawget(c,'_parent_with_init') then -- constructor exists
            local res = call_ctor(c,obj,...)
            if res then -- _if_ a ctor returns a value, it becomes the object...
                obj = res
                setmetatable(obj,c)
            end
        end

        if base and rawget(base,'_post_init') then
            base._post_init(obj)
        end

        return obj
    end
    -- Call Class.catch to set a handler for methods/properties not found in the class!
    c.catch = function(self, handler)
        if type(self) == "function" then
            -- called using . instead of :
            handler = self
        end
        c._handler = handler
        mt.__index = handler
    end
    c.is_a = is_a
    c.class_of = class_of
    c.cast = cast
    c._class = c

    if not rawget(c,'__tostring') then
        c.__tostring = _class_tostring
    end

    return c
end

--- create a new class, derived from a given base class.
-- Supporting two class creation syntaxes:
-- either `Name = class(base)` or `class.Name(base)`.
-- The first form returns the class directly and does not set its `_name`.
-- The second form creates a variable `Name` in the current environment set
-- to the class, and also sets `_name`.
-- @function class
-- @param base optional base class
-- @param c_arg optional parameter to class constructor
-- @param c optional table to be used as class
local class
class = setmetatable({},{
    __call = function(fun,...)
        return _class(...)
    end,
    __index = function(tbl,key)
        if key == 'class' then
            io.stderr:write('require("pl.class").class is deprecated. Use require("pl.class")\n')
            return class
        end
        compat = compat or require 'pl.compat'
        local env = compat.getfenv(2)
        return function(...)
            local c = _class(...)
            c._name = key
            rawset(env,key,c)
            return c
        end
    end
})

class.properties = class()

function class.properties._class_init(klass)
    klass.__index = function(t,key)
        -- normal class lookup!
        local v = klass[key]
        if v then return v end
        -- is it a getter?
        v = rawget(klass,'get_'..key)
        if v then
            return v(t)
        end
        -- is it a field?
        return rawget(t,'_'..key)
    end
    klass.__newindex = function (t,key,value)
        -- if there's a setter, use that, otherwise directly set table
        local p = 'set_'..key
        local setter = klass[p]
        if setter then
            setter(t,value)
        else
            rawset(t,key,value)
        end
    end
end


return class

