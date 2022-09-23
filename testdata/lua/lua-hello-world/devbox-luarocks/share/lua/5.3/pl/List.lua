--- Python-style list class.
--
-- **Please Note**: methods that change the list will return the list.
-- This is to allow for method chaining, but please note that `ls = ls:sort()`
-- does not mean that a new copy of the list is made. In-place (mutable) methods
-- are marked as returning 'the list' in this documentation.
--
-- See the Guide for further @{02-arrays.md.Python_style_Lists|discussion}
--
-- See <a href="http://www.python.org/doc/current/tut/tut.html">http://www.python.org/doc/current/tut/tut.html</a>, section 5.1
--
-- **Note**: The comments before some of the functions are from the Python docs
-- and contain Python code.
--
-- Written for Lua version Nick Trout 4.0; Redone for Lua 5.1, Steve Donovan.
--
-- Dependencies: `pl.utils`, `pl.tablex`, `pl.class`
-- @classmod pl.List
-- @pragma nostrip

local tinsert,tremove,concat,tsort = table.insert,table.remove,table.concat,table.sort
local setmetatable, getmetatable,type,tostring,string = setmetatable,getmetatable,type,tostring,string
local tablex = require 'pl.tablex'
local filter,imap,imap2,reduce,transform,tremovevalues = tablex.filter,tablex.imap,tablex.imap2,tablex.reduce,tablex.transform,tablex.removevalues
local tsub = tablex.sub
local utils = require 'pl.utils'
local class = require 'pl.class'

local array_tostring,split,assert_arg,function_arg = utils.array_tostring,utils.split,utils.assert_arg,utils.function_arg
local normalize_slice = tablex._normalize_slice

-- metatable for our list and map objects has already been defined..
local Multimap = utils.stdmt.MultiMap
local List = utils.stdmt.List

local iter

class(nil,nil,List)

-- we want the result to be _covariant_, i.e. t must have type of obj if possible
local function makelist (t,obj)
    local klass = List
    if obj then
        klass = getmetatable(obj)
    end
    return setmetatable(t,klass)
end

local function simple_table(t)
    return type(t) == 'table' and not getmetatable(t) and #t > 0
end

function List._create (src)
    if simple_table(src) then return src end
end

function List:_init (src)
    if self == src then return end -- existing table used as self!
    if src then
        for v in iter(src) do
            tinsert(self,v)
        end
    end
end

--- Create a new list. Can optionally pass a table;
-- passing another instance of List will cause a copy to be created;
-- this will return a plain table with an appropriate metatable.
-- we pass anything which isn't a simple table to iterate() to work out
-- an appropriate iterator
-- @see List.iterate
-- @param[opt] t An optional list-like table
-- @return a new List
-- @usage ls = List();  ls = List {1,2,3,4}
-- @function List.new

List.new = List

--- Make a copy of an existing list.
-- The difference from a plain 'copy constructor' is that this returns
-- the actual List subtype.
function List:clone()
    local ls = makelist({},self)
    ls:extend(self)
    return ls
end

--- Add an item to the end of the list.
-- @param i An item
-- @return the list
function List:append(i)
    tinsert(self,i)
    return self
end

List.push = tinsert

--- Extend the list by appending all the items in the given list.
-- equivalent to 'a[len(a):] = L'.
-- @tparam List L Another List
-- @return the list
function List:extend(L)
    assert_arg(1,L,'table')
    for i = 1,#L do tinsert(self,L[i]) end
    return self
end

--- Insert an item at a given position. i is the index of the
-- element before which to insert.
-- @int i index of element before whichh to insert
-- @param x A data item
-- @return the list
function List:insert(i, x)
    assert_arg(1,i,'number')
    tinsert(self,i,x)
    return self
end

--- Insert an item at the begining of the list.
-- @param x a data item
-- @return the list
function List:put (x)
    return self:insert(1,x)
end

--- Remove an element given its index.
-- (equivalent of Python's del s[i])
-- @int i the index
-- @return the list
function List:remove (i)
    assert_arg(1,i,'number')
    tremove(self,i)
    return self
end

--- Remove the first item from the list whose value is given.
-- (This is called 'remove' in Python; renamed to avoid confusion
-- with table.remove)
-- Return nil if there is no such item.
-- @param x A data value
-- @return the list
function List:remove_value(x)
    for i=1,#self do
        if self[i]==x then tremove(self,i) return self end
    end
    return self
 end

--- Remove the item at the given position in the list, and return it.
-- If no index is specified, a:pop() returns the last item in the list.
-- The item is also removed from the list.
-- @int[opt] i An index
-- @return the item
function List:pop(i)
    if not i then i = #self end
    assert_arg(1,i,'number')
    return tremove(self,i)
end

List.get = List.pop

--- Return the index in the list of the first item whose value is given.
-- Return nil if there is no such item.
-- @function List:index
-- @param x A data value
-- @int[opt=1] idx where to start search
-- @return the index, or nil if not found.

local tfind = tablex.find
List.index = tfind

--- Does this list contain the value?
-- @param x A data value
-- @return true or false
function List:contains(x)
    return tfind(self,x) and true or false
end

--- Return the number of times value appears in the list.
-- @param x A data value
-- @return number of times x appears
function List:count(x)
    local cnt=0
    for i=1,#self do
        if self[i]==x then cnt=cnt+1 end
    end
    return cnt
end

--- Sort the items of the list, in place.
-- @func[opt='<'] cmp an optional comparison function
-- @return the list
function List:sort(cmp)
    if cmp then cmp = function_arg(1,cmp) end
    tsort(self,cmp)
    return self
end

--- Return a sorted copy of this list.
-- @func[opt='<'] cmp an optional comparison function
-- @return a new list
function List:sorted(cmp)
    return List(self):sort(cmp)
end

--- Reverse the elements of the list, in place.
-- @return the list
function List:reverse()
    local t = self
    local n = #t
    for i = 1,n/2 do
        t[i],t[n] = t[n],t[i]
        n = n - 1
    end
    return self
end

--- Return the minimum and the maximum value of the list.
-- @return minimum value
-- @return maximum value
function List:minmax()
    local vmin,vmax = 1e70,-1e70
    for i = 1,#self do
        local v = self[i]
        if v < vmin then vmin = v end
        if v > vmax then vmax = v end
    end
    return vmin,vmax
end

--- Emulate list slicing.  like  'list[first:last]' in Python.
-- If first or last are negative then they are relative to the end of the list
-- eg. slice(-2) gives last 2 entries in a list, and
-- slice(-4,-2) gives from -4th to -2nd
-- @param first An index
-- @param last An index
-- @return a new List
function List:slice(first,last)
    return tsub(self,first,last)
end

--- Empty the list.
-- @return the list
function List:clear()
    for i=1,#self do tremove(self) end
    return self
end

local eps = 1.0e-10

--- Emulate Python's range(x) function.
-- Include it in List table for tidiness
-- @int start A number
-- @int[opt] finish A number greater than start; if absent,
-- then start is 1 and finish is start
-- @int[opt=1] incr an increment (may be less than 1)
-- @return a List from start .. finish
-- @usage List.range(0,3) == List{0,1,2,3}
-- @usage List.range(4) = List{1,2,3,4}
-- @usage List.range(5,1,-1) == List{5,4,3,2,1}
function List.range(start,finish,incr)
    if not finish then
        finish = start
        start = 1
    end
    if incr then
    assert_arg(3,incr,'number')
    if math.ceil(incr) ~= incr then finish = finish + eps end
    else
        incr = 1
    end
    assert_arg(1,start,'number')
    assert_arg(2,finish,'number')
    local t = List()
    for i=start,finish,incr do tinsert(t,i) end
    return t
end

--- list:len() is the same as #list.
function List:len()
    return #self
end

-- Extended operations --

--- Remove a subrange of elements.
-- equivalent to 'del s[i1:i2]' in Python.
-- @int i1 start of range
-- @int i2 end of range
-- @return the list
function List:chop(i1,i2)
    return tremovevalues(self,i1,i2)
end

--- Insert a sublist into a list
-- equivalent to 's[idx:idx] = list' in Python
-- @int idx index
-- @tparam List list list to insert
-- @return the list
-- @usage  l = List{10,20}; l:splice(2,{21,22});  assert(l == List{10,21,22,20})
function List:splice(idx,list)
    assert_arg(1,idx,'number')
    idx = idx - 1
    local i = 1
    for v in iter(list) do
        tinsert(self,i+idx,v)
        i = i + 1
    end
    return self
end

--- General slice assignment s[i1:i2] = seq.
-- @int i1  start index
-- @int i2  end index
-- @tparam List seq a list
-- @return the list
function List:slice_assign(i1,i2,seq)
    assert_arg(1,i1,'number')
    assert_arg(1,i2,'number')
    i1,i2 = normalize_slice(self,i1,i2)
    if i2 >= i1 then self:chop(i1,i2) end
    self:splice(i1,seq)
    return self
end

--- Concatenation operator.
-- @within metamethods
-- @tparam List L another List
-- @return a new list consisting of the list with the elements of the new list appended
function List:__concat(L)
    assert_arg(1,L,'table')
    local ls = self:clone()
    ls:extend(L)
    return ls
end

--- Equality operator ==.  True iff all elements of two lists are equal.
-- @within metamethods
-- @tparam List L another List
-- @return true or false
function List:__eq(L)
    if #self ~= #L then return false end
    for i = 1,#self do
        if self[i] ~= L[i] then return false end
    end
    return true
end

--- Join the elements of a list using a delimiter.
-- This method uses tostring on all elements.
-- @string[opt=''] delim a delimiter string, can be empty.
-- @return a string
function List:join (delim)
    delim = delim or ''
    assert_arg(1,delim,'string')
    return concat(array_tostring(self),delim)
end

--- Join a list of strings. <br>
-- Uses `table.concat` directly.
-- @function List:concat
-- @string[opt=''] delim a delimiter
-- @return a string
List.concat = concat

local function tostring_q(val)
    local s = tostring(val)
    if type(val) == 'string' then
        s = '"'..s..'"'
    end
    return s
end

--- How our list should be rendered as a string. Uses join().
-- @within metamethods
-- @see List:join
function List:__tostring()
    return '{'..self:join(',',tostring_q)..'}'
end

--- Call the function on each element of the list.
-- @func fun a function or callable object
-- @param ... optional values to pass to function
function List:foreach (fun,...)
    fun = function_arg(1,fun)
    for i = 1,#self do
        fun(self[i],...)
    end
end

local function lookup_fun (obj,name)
    local f = obj[name]
    if not f then error(type(obj).." does not have method "..name,3) end
    return f
end

--- Call the named method on each element of the list.
-- @string name the method name
-- @param ... optional values to pass to function
function List:foreachm (name,...)
    for i = 1,#self do
        local obj = self[i]
        local f = lookup_fun(obj,name)
        f(obj,...)
    end
end

--- Create a list of all elements which match a function.
-- @func fun a boolean function
-- @param[opt] arg optional argument to be passed as second argument of the predicate
-- @return a new filtered list.
function List:filter (fun,arg)
    return makelist(filter(self,fun,arg),self)
end

--- Split a string using a delimiter.
-- @string s the string
-- @string[opt] delim the delimiter (default spaces)
-- @return a List of strings
-- @see pl.utils.split
function List.split (s,delim)
    assert_arg(1,s,'string')
    return makelist(split(s,delim))
end

--- Apply a function to all elements.
-- Any extra arguments will be passed to the function.
-- @func fun a function of at least one argument
-- @param ... arbitrary extra arguments.
-- @return a new list: {f(x) for x in self}
-- @usage List{'one','two'}:map(string.upper) == {'ONE','TWO'}
-- @see pl.tablex.imap
function List:map (fun,...)
    return makelist(imap(fun,self,...),self)
end

--- Apply a function to all elements, in-place.
-- Any extra arguments are passed to the function.
-- @func fun A function that takes at least one argument
-- @param ... arbitrary extra arguments.
-- @return the list.
function List:transform (fun,...)
    transform(fun,self,...)
    return self
end

--- Apply a function to elements of two lists.
-- Any extra arguments will be passed to the function
-- @func fun a function of at least two arguments
-- @tparam List ls another list
-- @param ... arbitrary extra arguments.
-- @return a new list: {f(x,y) for x in self, for x in arg1}
-- @see pl.tablex.imap2
function List:map2 (fun,ls,...)
    return makelist(imap2(fun,self,ls,...),self)
end

--- apply a named method to all elements.
-- Any extra arguments will be passed to the method.
-- @string name name of method
-- @param ... extra arguments
-- @return a new list of the results
-- @see pl.seq.mapmethod
function List:mapm (name,...)
    local res = {}
    for i = 1,#self do
      local val = self[i]
      local fn = lookup_fun(val,name)
      res[i] = fn(val,...)
    end
    return makelist(res,self)
end

local function composite_call (method,f)
    return function(self,...)
        return self[method](self,f,...)
    end
end

function List.default_map_with(T)
    return function(self,name)
        local m
        if T then
            local f = lookup_fun(T,name)
            m = composite_call('map',f)
        else
            m = composite_call('mapn',name)
        end
        getmetatable(self)[name] = m -- and cache..
        return m
    end
end

List.default_map = List.default_map_with

--- 'reduce' a list using a binary function.
-- @func fun a function of two arguments
-- @return result of the function
-- @see pl.tablex.reduce
function List:reduce (fun)
    return reduce(fun,self)
end

--- Partition a list using a classifier function.
-- The function may return nil, but this will be converted to the string key '<nil>'.
-- @func fun a function of at least one argument
-- @param ... will also be passed to the function
-- @treturn MultiMap a table where the keys are the returned values, and the values are Lists
-- of values where the function returned that key.
-- @see pl.MultiMap
function List:partition (fun,...)
    fun = function_arg(1,fun)
    local res = {}
    for i = 1,#self do
        local val = self[i]
        local klass = fun(val,...)
        if klass == nil then klass = '<nil>' end
        if not res[klass] then res[klass] = List() end
        res[klass]:append(val)
    end
    return setmetatable(res,Multimap)
end

--- return an iterator over all values.
function List:iter ()
    return iter(self)
end

--- Create an iterator over a seqence.
-- This captures the Python concept of 'sequence'.
-- For tables, iterates over all values with integer indices.
-- @param seq a sequence; a string (over characters), a table, a file object (over lines) or an iterator function
-- @usage for x in iterate {1,10,22,55} do io.write(x,',') end ==> 1,10,22,55
-- @usage for ch in iterate 'help' do do io.write(ch,' ') end ==> h e l p
function List.iterate(seq)
    if type(seq) == 'string' then
        local idx = 0
        local n = #seq
        local sub = string.sub
        return function ()
            idx = idx + 1
            if idx > n then return nil
            else
                return sub(seq,idx,idx)
            end
        end
    elseif type(seq) == 'table' then
        local idx = 0
        local n = #seq
        return function()
            idx = idx + 1
            if idx > n then return nil
            else
                return seq[idx]
            end
        end
    elseif type(seq) == 'function' then
        return seq
    elseif type(seq) == 'userdata' and io.type(seq) == 'file' then
        return seq:lines()
    end
end
iter = List.iterate

return List

