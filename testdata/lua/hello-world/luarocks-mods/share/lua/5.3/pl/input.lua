--- Iterators for extracting words or numbers from an input source.
--
--    require 'pl'
--    local total,n = seq.sum(input.numbers())
--    print('average',total/n)
--
-- _source_ is defined as a string or a file-like object (i.e. has a read() method which returns the next line)
--
-- See @{06-data.md.Reading_Unstructured_Text_Data|here}
--
-- Dependencies: `pl.utils`
-- @module pl.input
local strfind = string.find
local strsub = string.sub
local strmatch = string.match
local utils = require 'pl.utils'
local unpack = utils.unpack
local pairs,type,tonumber = pairs,type,tonumber
local patterns = utils.patterns
local io = io

local input = {}

--- create an iterator over all tokens.
-- based on allwords from PiL, 7.1
-- @func getter any function that returns a line of text
-- @string pattern
-- @string[opt] fn  Optionally can pass a function to process each token as it's found.
-- @return an iterator
function input.alltokens (getter,pattern,fn)
    local line = getter()  -- current line
    local pos = 1           -- current position in the line
    return function ()      -- iterator function
        while line do         -- repeat while there are lines
          local s, e = strfind(line, pattern, pos)
          if s then           -- found a word?
            pos = e + 1       -- next position is after this token
            local res = strsub(line, s, e)     -- return the token
            if fn then res = fn(res) end
            return res
          else
            line = getter()  -- token not found; try next line
            pos = 1           -- restart from first position
          end
        end
        return nil            -- no more lines: end of traversal
   end
end
local alltokens = input.alltokens

-- question: shd this _split_ a string containing line feeds?

--- create a function which grabs the next value from a source. If the source is a string, then the getter
-- will return the string and thereafter return nil. If not specified then the source is assumed to be stdin.
-- @param f a string or a file-like object (i.e. has a read() method which returns the next line)
-- @return a getter function
function input.create_getter(f)
    if f then
        if type(f) == 'string' then
            local ls = utils.split(f,'\n')
            local i,n = 0,#ls
            return function()
                i = i + 1
                if i > n then return nil end
                return ls[i]
            end
        else
            -- anything that supports the read() method!
            if not f.read then error('not a file-like object') end
            return function() return f:read() end
        end
    else
        return io.read  -- i.e. just read from stdin
    end
end

--- generate a sequence of numbers from a source.
-- @param f A source
-- @return An iterator
function input.numbers(f)
    return alltokens(input.create_getter(f),
        '('..patterns.FLOAT..')',tonumber)
end

--- generate a sequence of words from a source.
-- @param f A source
-- @return An iterator
function input.words(f)
    return alltokens(input.create_getter(f),"%w+")
end

local function apply_tonumber (no_fail,...)
    local args = {...}
    for i = 1,#args do
        local n = tonumber(args[i])
        if  n == nil then
            if not no_fail then return nil,args[i] end
        else
            args[i] = n
        end
    end
    return args
end

--- parse an input source into fields.
-- By default, will fail if it cannot convert a field to a number.
-- @param ids a list of field indices, or a maximum field index
-- @string delim delimiter to parse fields (default space)
-- @param f a source @see create_getter
-- @tab opts option table, `{no_fail=true}`
-- @return an iterator with the field values
-- @usage for x,y in fields {2,3} do print(x,y) end -- 2nd and 3rd fields from stdin
function input.fields (ids,delim,f,opts)
  local sep
  local s
  local getter = input.create_getter(f)
  local no_fail = opts and opts.no_fail
  local no_convert = opts and opts.no_convert
  if not delim or delim == ' ' then
      delim = '%s'
      sep = '%s+'
      s = '%s*'
  else
      sep = delim
      s = ''
  end
  local max_id = 0
  if type(ids) == 'table' then
    for i,id in pairs(ids) do
      if id > max_id then max_id = id end
    end
  else
    max_id = ids
    ids = {}
    for i = 1,max_id do ids[#ids+1] = i end
  end
  local pat = '[^'..delim..']*'
  local k = 1
  for i = 1,max_id do
    if ids[k] == i then
      k = k + 1
      s = s..'('..pat..')'
    else
      s = s..pat
    end
    if i < max_id then
      s = s..sep
    end
  end
  local linecount = 1
  return function()
    local line,results,err
    repeat
        line = getter()
        linecount = linecount + 1
        if not line then return nil end
        if no_convert then
            results = {strmatch(line,s)}
        else
            results,err = apply_tonumber(no_fail,strmatch(line,s))
            if not results then
                utils.quit("line "..(linecount-1)..": cannot convert '"..err.."' to number")
            end
        end
    until #results > 0
    return unpack(results)
  end
end

return input

