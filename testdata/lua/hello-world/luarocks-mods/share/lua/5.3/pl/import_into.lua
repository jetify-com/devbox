--------------
-- PL loader, for loading all PL libraries, only on demand.
-- Whenever a module is implicitly accesssed, the table will have the module automatically injected.
-- (e.g. `_ENV.tablex`)
-- then that module is dynamically loaded. The submodules are all brought into
-- the table that is provided as the argument, or returned in a new table.
-- If a table is provided, that table's metatable is clobbered, but the values are not.
-- This module returns a single function, which is passed the environment.
-- If this is `true`, then return a 'shadow table' as the module
-- See @{01-introduction.md.To_Inject_or_not_to_Inject_|the Guide}

-- @module pl.import_into

return function(env)
    local mod
    if env == true then
        mod = {}
        env = {}
    end
    local env = env or {}

    local modules = {
        utils = true,path=true,dir=true,tablex=true,stringio=true,sip=true,
        input=true,seq=true,lexer=true,stringx=true,
        config=true,pretty=true,data=true,func=true,text=true,
        operator=true,lapp=true,array2d=true,
        comprehension=true,xml=true,types=true,
        test = true, app = true, file = true, class = true,
        luabalanced = true, permute = true, template = true,
        url = true, compat = true,
        -- classes --
        List = true, Map = true, Set = true,
        OrderedMap = true, MultiMap = true, Date = true,
    }
    rawset(env,'utils',require 'pl.utils')

    for name,klass in pairs(env.utils.stdmt) do
        klass.__index = function(t,key)
            return require ('pl.'..name)[key]
        end;
    end

    -- ensure that we play nice with libraries that also attach a metatable
    -- to the global table; always forward to a custom __index if we don't
    -- match

    local _hook,_prev_index
    local gmt = {}
    local prevenvmt = getmetatable(env)
    if prevenvmt then
        _prev_index = prevenvmt.__index
        if prevenvmt.__newindex then
            gmt.__newindex = prevenvmt.__newindex
        end
    end

    function gmt.hook(handler)
        _hook = handler
    end

    function gmt.__index(t,name)
        local found = modules[name]
        -- either true, or the name of the module containing this class.
        -- either way, we load the required module and make it globally available.
        if found then
            -- e..g pretty.dump causes pl.pretty to become available as 'pretty'
            rawset(env,name,require('pl.'..name))
            return env[name]
        else
            local res
            if _hook then
                res = _hook(t,name)
                if res then return res end
            end
            if _prev_index then
                return _prev_index(t,name)
            end
        end
    end

    if mod then
        function gmt.__newindex(t,name,value)
            mod[name] = value
            rawset(t,name,value)
        end
    end

    setmetatable(env,gmt)

    return env,mod or env
end
