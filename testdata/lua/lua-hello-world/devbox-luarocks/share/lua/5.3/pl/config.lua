--- Reads configuration files into a Lua table.
--  Understands INI files, classic Unix config files, and simple
-- delimited columns of values. See @{06-data.md.Reading_Configuration_Files|the Guide}
--
--    # test.config
--    # Read timeout in seconds
--    read.timeout=10
--    # Write timeout in seconds
--    write.timeout=5
--    #acceptable ports
--    ports = 1002,1003,1004
--
--    -- readconfig.lua
--    local config = require 'config'
--    local t = config.read 'test.config'
--    print(pretty.write(t))
--
--    ### output #####
--    {
--      ports = {
--        1002,
--        1003,
--        1004
--      },
--      write_timeout = 5,
--      read_timeout = 10
--    }
--
-- @module pl.config

local type,tonumber,ipairs,io, table = _G.type,_G.tonumber,_G.ipairs,_G.io,_G.table

local function split(s,re)
    local res = {}
    local t_insert = table.insert
    re = '[^'..re..']+'
    for k in s:gmatch(re) do t_insert(res,k) end
    return res
end

local function strip(s)
    return s:gsub('^%s+',''):gsub('%s+$','')
end

local function strip_quotes (s)
    return s:gsub("['\"](.*)['\"]",'%1')
end

local config = {}

--- like io.lines(), but allows for lines to be continued with '\'.
-- @param file a file-like object (anything where read() returns the next line) or a filename.
-- Defaults to stardard input.
-- @return an iterator over the lines, or nil
-- @return error 'not a file-like object' or 'file is nil'
function config.lines(file)
    local f,openf,err
    local line = ''
    if type(file) == 'string' then
        f,err = io.open(file,'r')
        if not f then return nil,err end
        openf = true
    else
        f = file or io.stdin
        if not file.read then return nil, 'not a file-like object' end
    end
    if not f then return nil, 'file is nil' end
    return function()
        local l = f:read()
        while l do
            -- only for non-blank lines that don't begin with either ';' or '#'
            if l:match '%S' and not l:match '^%s*[;#]' then
                -- does the line end with '\'?
                local i = l:find '\\%s*$'
                if i then -- if so,
                    line = line..l:sub(1,i-1)
                elseif line == '' then
                    return l
                else
                    l = line..l
                    line = ''
                    return l
                end
            end
            l = f:read()
        end
        if openf then f:close() end
    end
end

--- read a configuration file into a table
-- @param file either a file-like object or a string, which must be a filename
-- @tab[opt] cnfg a configuration table that may contain these fields:
--
--  * `smart`  try to deduce what kind of config file we have (default false)
--  * `variabilize` make names into valid Lua identifiers (default true)
--  * `convert_numbers` try to convert values into numbers (default true)
--  * `trim_space` ensure that there is no starting or trailing whitespace with values (default true)
--  * `trim_quotes` remove quotes from strings (default false)
--  * `list_delim` delimiter to use when separating columns (default ',')
--  * `keysep` separator between key and value pairs (default '=')
--
-- @return a table containing items, or `nil`
-- @return error message (same as @{config.lines}
function config.read(file,cnfg)
    local auto

    local iter,err = config.lines(file)
    if not iter then return nil,err end
    local line = iter()
    cnfg = cnfg or {}
    if cnfg.smart then
        auto = true
        if line:match '^[^=]+=' then
            cnfg.keysep = '='
        elseif line:match '^[^:]+:' then
            cnfg.keysep = ':'
            cnfg.list_delim = ':'
        elseif line:match '^%S+%s+' then
            cnfg.keysep = ' '
            -- more than two columns assume that it's a space-delimited list
            -- cf /etc/fstab with /etc/ssh/ssh_config
            if line:match '^%S+%s+%S+%s+%S+' then
                cnfg.list_delim = ' '
            end
            cnfg.variabilize = false
        end
    end


    local function check_cnfg (var,def)
        local val = cnfg[var]
        if val == nil then return def else return val end
    end

    local initial_digits = '^[%d%+%-]'
    local t = {}
    local top_t = t
    local variabilize = check_cnfg ('variabilize',true)
    local list_delim = check_cnfg('list_delim',',')
    local convert_numbers = check_cnfg('convert_numbers',true)
    local convert_boolean = check_cnfg('convert_boolean',false)
    local trim_space = check_cnfg('trim_space',true)
    local trim_quotes = check_cnfg('trim_quotes',false)
    local ignore_assign = check_cnfg('ignore_assign',false)
    local keysep = check_cnfg('keysep','=')
    local keypat = keysep == ' ' and '%s+' or '%s*'..keysep..'%s*'
    if list_delim == ' ' then list_delim = '%s+' end

    local function process_name(key)
        if variabilize then
            key = key:gsub('[^%w]','_')
        end
        return key
    end

    local function process_value(value)
        if list_delim and value:find(list_delim) then
            value = split(value,list_delim)
            for i,v in ipairs(value) do
                value[i] = process_value(v)
            end
        elseif convert_numbers and value:find(initial_digits) then
            local val = tonumber(value)
            if not val and value:match ' kB$' then
                value = value:gsub(' kB','')
                val = tonumber(value)
            end
            if val then value = val end
        elseif convert_boolean and value == 'true' then
           return true
        elseif convert_boolean and value == 'false' then
           return false
        end
        if type(value) == 'string' then
           if trim_space then value = strip(value) end
           if not trim_quotes and auto and value:match '^"' then
                trim_quotes = true
            end
           if trim_quotes then value = strip_quotes(value) end
        end
        return value
    end

    while line do
        if line:find('^%[') then -- section!
            local section = process_name(line:match('%[([^%]]+)%]'))
            t = top_t
            t[section] = {}
            t = t[section]
        else
            line = line:gsub('^%s*','')
            local i1,i2 = line:find(keypat)
            if i1 and not ignore_assign then -- key,value assignment
                local key = process_name(line:sub(1,i1-1))
                local value = process_value(line:sub(i2+1))
                t[key] = value
            else -- a plain list of values...
                t[#t+1] = process_value(line)
            end
        end
        line = iter()
    end
    return top_t
end

return config
