--- Manipulating iterators as sequences.
-- See @{07-functional.md.Sequences|The Guide}
--
-- Dependencies: `pl.utils`, `pl.types`, `debug`
-- @module pl.seq

local next,assert,pairs,tonumber,type,setmetatable = next,assert,pairs,tonumber,type,setmetatable
local strfind,format = string.find,string.format
local mrandom = math.random
local tsort,tappend = table.sort,table.insert
local io = io
local utils = require 'pl.utils'
local callable = require 'pl.types'.is_callable
local function_arg = utils.function_arg
local assert_arg = utils.assert_arg
local debug = require 'debug'

local seq = {}

-- given a number, return a function(y) which returns true if y > x
-- @param x a number
function seq.greater_than(x)
  return function(v)
    return tonumber(v) > x
  end
end

-- given a number, returns a function(y) which returns true if y < x
-- @param x a number
function seq.less_than(x)
  return function(v)
    return tonumber(v) < x
  end
end

-- given any value, return a function(y) which returns true if y == x
-- @param x a value
function seq.equal_to(x)
  if type(x) == "number" then
    return function(v)
      return tonumber(v) == x
    end
  else
    return function(v)
      return v == x
    end
  end
end

--- given a string, return a function(y) which matches y against the string.
-- @param s a string
function seq.matching(s)
  return function(v)
     return strfind(v,s)
  end
end

local nexti

--- sequence adaptor for a table.   Note that if any generic function is
-- passed a table, it will automatically use seq.list()
-- @param t a list-like table
-- @usage sum(list(t)) is the sum of all elements of t
-- @usage for x in list(t) do...end
function seq.list(t)
  assert_arg(1,t,'table')
  if not nexti then
    nexti = ipairs{}
  end
  local key,value = 0
  return function()
    key,value = nexti(t,key)
    return value
  end
end

--- return the keys of the table.
-- @param t an arbitrary table
-- @return iterator over keys
function seq.keys(t)
  assert_arg(1,t,'table')
  local key
  return function()
    key = next(t,key)
    return key
  end
end

local list = seq.list
local function default_iter(iter)
  if type(iter) == 'table' then return list(iter)
  else return iter end
end

seq.iter = default_iter

--- create an iterator over a numerical range. Like the standard Python function xrange.
-- @param start a number
-- @param finish a number greater than start
function seq.range(start,finish)
  local i = start - 1
  return function()
      i = i + 1
      if i > finish then return nil
      else return i end
  end
end

-- count the number of elements in the sequence which satisfy the predicate
-- @param iter a sequence
-- @param condn a predicate function (must return either true or false)
-- @param optional argument to be passed to predicate as second argument.
-- @return count
function seq.count(iter,condn,arg)
  local i = 0
  seq.foreach(iter,function(val)
        if condn(val,arg) then i = i + 1 end
  end)
  return i
end

--- return the minimum and the maximum value of the sequence.
-- @param iter a sequence
-- @return minimum value
-- @return maximum value
function seq.minmax(iter)
  local vmin,vmax = 1e70,-1e70
  for v in default_iter(iter) do
    v = tonumber(v)
    if v < vmin then vmin = v end
    if v > vmax then vmax = v end
  end
  return vmin,vmax
end

--- return the sum and element count of the sequence.
-- @param iter a sequence
-- @param fn an optional function to apply to the values
function seq.sum(iter,fn)
  local s = 0
  local i = 0
  for v in default_iter(iter) do
    if fn then v = fn(v) end
    s = s + v
    i = i + 1
  end
  return s,i
end

--- create a table from the sequence. (This will make the result a List.)
-- @param iter a sequence
-- @return a List
-- @usage copy(list(ls)) is equal to ls
-- @usage copy(list {1,2,3}) == List{1,2,3}
function seq.copy(iter)
    local res,k = {},1
    for v in default_iter(iter) do
        res[k] = v
        k = k + 1
    end
    setmetatable(res, require('pl.List'))
    return res
end

--- create a table of pairs from the double-valued sequence.
-- @param iter a double-valued sequence
-- @param i1 used to capture extra iterator values
-- @param i2 as with pairs & ipairs
-- @usage copy2(ipairs{10,20,30}) == {{1,10},{2,20},{3,30}}
-- @return a list-like table
function seq.copy2 (iter,i1,i2)
    local res,k = {},1
    for v1,v2 in iter,i1,i2 do
        res[k] = {v1,v2}
        k = k + 1
    end
    return res
end

--- create a table of 'tuples' from a multi-valued sequence.
-- A generalization of copy2 above
-- @param iter a multiple-valued sequence
-- @return a list-like table
function seq.copy_tuples (iter)
    iter = default_iter(iter)
    local res = {}
    local row = {iter()}
    while #row > 0 do
        tappend(res,row)
        row = {iter()}
    end
    return res
end

--- return an iterator of random numbers.
-- @param n the length of the sequence
-- @param l same as the first optional argument to math.random
-- @param u same as the second optional argument to math.random
-- @return a sequence
function seq.random(n,l,u)
  local rand
  assert(type(n) == 'number')
  if u then
     rand = function() return mrandom(l,u) end
  elseif l then
     rand = function() return mrandom(l) end
  else
     rand = mrandom
  end

  return function()
     if n == 0 then return nil
     else
       n = n - 1
       return rand()
     end
  end
end

--- return an iterator to the sorted elements of a sequence.
-- @param iter a sequence
-- @param comp an optional comparison function (comp(x,y) is true if x < y)
function seq.sort(iter,comp)
    local t = seq.copy(iter)
    tsort(t,comp)
    return list(t)
end

--- return an iterator which returns elements of two sequences.
-- @param iter1 a sequence
-- @param iter2 a sequence
-- @usage for x,y in seq.zip(ls1,ls2) do....end
function seq.zip(iter1,iter2)
    iter1 = default_iter(iter1)
    iter2 = default_iter(iter2)
    return function()
        return iter1(),iter2()
    end
end

--- Makes a table where the key/values are the values and value counts of the sequence.
-- This version works with 'hashable' values like strings and numbers.
-- `pl.tablex.count_map` is more general.
-- @param iter a sequence
-- @return a map-like table
-- @return a table
-- @see pl.tablex.count_map
function seq.count_map(iter)
    local t = {}
    local v
    for s in default_iter(iter) do
        v = t[s]
        if v then t[s] = v + 1
        else t[s] = 1 end
    end
    return setmetatable(t, require('pl.Map'))
end

-- given a sequence, return all the unique values in that sequence.
-- @param iter a sequence
-- @param returns_table true if we return a table, not a sequence
-- @return a sequence or a table; defaults to a sequence.
function seq.unique(iter,returns_table)
    local t = seq.count_map(iter)
    local res,k = {},1
    for key in pairs(t) do res[k] = key; k = k + 1 end
    table.sort(res)
    if returns_table then
        return res
    else
        return list(res)
    end
end

--- print out a sequence iter with a separator.
-- @param iter a sequence
-- @param sep the separator (default space)
-- @param nfields maximum number of values per line (default 7)
-- @param fmt optional format function for each value
function seq.printall(iter,sep,nfields,fmt)
  local write = io.write
  if not sep then sep = ' ' end
  if not nfields then
      if sep == '\n' then nfields = 1e30
      else nfields = 7 end
  end
  if fmt then
    local fstr = fmt
    fmt = function(v) return format(fstr,v) end
  end
  local k = 1
  for v in default_iter(iter) do
     if fmt then v = fmt(v) end
     if k < nfields then
       write(v,sep)
       k = k + 1
    else
       write(v,'\n')
       k = 1
    end
  end
  write '\n'
end

-- return an iterator running over every element of two sequences (concatenation).
-- @param iter1 a sequence
-- @param iter2 a sequence
function seq.splice(iter1,iter2)
  iter1 = default_iter(iter1)
  iter2 = default_iter(iter2)
  local iter = iter1
  return function()
    local ret = iter()
    if ret == nil then
      if iter == iter1 then
        iter = iter2
        return iter()
      else return nil end
   else
       return  ret
   end
 end
end

--- return a sequence where every element of a sequence has been transformed
-- by a function. If you don't supply an argument, then the function will
-- receive both values of a double-valued sequence, otherwise behaves rather like
-- tablex.map.
-- @param fn a function to apply to elements; may take two arguments
-- @param iter a sequence of one or two values
-- @param arg optional argument to pass to function.
function seq.map(fn,iter,arg)
    fn = function_arg(1,fn)
    iter = default_iter(iter)
    return function()
        local v1,v2 = iter()
        if v1 == nil then return nil end
        return fn(v1,arg or v2) or false
    end
end

--- filter a sequence using a predicate function.
-- @param iter a sequence of one or two values
-- @param pred a boolean function; may take two arguments
-- @param arg optional argument to pass to function.
function seq.filter (iter,pred,arg)
    pred = function_arg(2,pred)
    return function ()
        local v1,v2
        while true do
            v1,v2 = iter()
            if v1 == nil then return nil end
            if pred(v1,arg or v2) then return v1,v2 end
        end
    end
end

--- 'reduce' a sequence using a binary function.
-- @func fn a function of two arguments
-- @param iter a sequence
-- @param initval optional initial value
-- @usage seq.reduce(operator.add,seq.list{1,2,3,4}) == 10
-- @usage seq.reduce('-',{1,2,3,4,5}) == -13
function seq.reduce (fn,iter,initval)
   fn = function_arg(1,fn)
   iter = default_iter(iter)
   local val = initval or iter()
   if val == nil then return nil end
   for v in iter do
       val = fn(val,v)
   end
   return val
end

--- take the first n values from the sequence.
-- @param iter a sequence of one or two values
-- @param n number of items to take
-- @return a sequence of at most n items
function seq.take (iter,n)
    iter = default_iter(iter)
    return function()
        if n < 1 then return end
        local val1,val2 = iter()
        if not val1 then return end
        n = n - 1
        return val1,val2
    end
end

--- skip the first n values of a sequence
-- @param iter a sequence of one or more values
-- @param n number of items to skip
function seq.skip (iter,n)
    n = n or 1
    for i = 1,n do
        if iter() == nil then return list{} end
    end
    return iter
end

--- a sequence with a sequence count and the original value.
-- enum(copy(ls)) is a roundabout way of saying ipairs(ls).
-- @param iter a single or double valued sequence
-- @return sequence of (i,v), i = 1..n and v is from iter.
function seq.enum (iter)
    local i = 0
    iter = default_iter(iter)
    return function  ()
        local val1,val2 = iter()
        if not val1 then return end
        i = i + 1
        return i,val1,val2
    end
end

--- map using a named method over a sequence.
-- @param iter a sequence
-- @param name the method name
-- @param arg1 optional first extra argument
-- @param arg2 optional second extra argument
function seq.mapmethod (iter,name,arg1,arg2)
    iter = default_iter(iter)
    return function()
        local val = iter()
        if not val then return end
        local fn = val[name]
        if not fn then error(type(val).." does not have method "..name) end
        return fn(val,arg1,arg2)
    end
end

--- a sequence of (last,current) values from another sequence.
--  This will return S(i-1),S(i) if given S(i)
-- @param iter a sequence
function seq.last (iter)
    iter = default_iter(iter)
    local val, l = iter(), nil
    if val == nil then return list{} end
    return function ()
        val,l = iter(),val
        if val == nil then return nil end
        return val,l
    end
end

--- call the function on each element of the sequence.
-- @param iter a sequence with up to 3 values
-- @param fn a function
function seq.foreach(iter,fn)
    fn = function_arg(2,fn)
    for i1,i2,i3 in default_iter(iter) do fn(i1,i2,i3) end
end

---------------------- Sequence Adapters ---------------------

local SMT

local function SW (iter,...)
    if callable(iter) then
        return setmetatable({iter=iter},SMT)
    else
        return iter,...
    end
end


-- can't directly look these up in seq because of the wrong argument order...
local map,reduce,mapmethod = seq.map, seq.reduce, seq.mapmethod
local overrides = {
    map = function(self,fun,arg)
        return map(fun,self,arg)
    end,
    reduce = function(self,fun,initval)
        return reduce(fun,self,initval)
    end
}

SMT = {
    __index = function (tbl,key)
        local fn = overrides[key] or seq[key]
        if fn then
            return function(sw,...) return SW(fn(sw.iter,...)) end
        else
            return function(sw,...) return SW(mapmethod(sw.iter,key,...)) end
        end
    end,
    __call = function (sw)
        return sw.iter()
    end,
}

setmetatable(seq,{
    __call = function(tbl,iter,extra)
        if not callable(iter) then
            if type(iter) == 'table' then iter = seq.list(iter)
            else return iter
            end
        end
        if extra then
            return setmetatable({iter=function()
                return iter(extra)
            end},SMT)
        else
            return setmetatable({iter=iter},SMT)
        end
    end
})

--- create a wrapped iterator over all lines in the file.
-- @param f either a filename, file-like object, or 'STDIN' (for standard input)
-- @param ... for Lua 5.2 only, optional format specifiers, as in `io.read`.
-- @return a sequence wrapper
function seq.lines (f,...)
    local iter,obj
    if f == 'STDIN' then
        f = io.stdin
    elseif type(f) == 'string' then
        iter,obj = io.lines(f,...)
    elseif not f.read then
        error("Pass either a string or a file-like object",2)
    end
    if not iter then
        iter,obj = f:lines(...)
    end
    if obj then -- LuaJIT version returns a function operating on a file
        local lines,file = iter,obj
        iter = function() return lines(file) end
    end
    return SW(iter)
end

function seq.import ()
    debug.setmetatable(function() end,{
        __index = function(tbl,key)
            local s = overrides[key] or seq[key]
            if s then return s
            else
                return function(s,...) return seq.mapmethod(s,key,...) end
            end
        end
    })
end

return seq
