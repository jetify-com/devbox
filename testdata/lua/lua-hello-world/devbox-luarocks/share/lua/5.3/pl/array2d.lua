--- Operations on two-dimensional arrays.
-- See @{02-arrays.md.Operations_on_two_dimensional_tables|The Guide}
--
-- The size of the arrays is determined by using the length operator `#` hence
-- the module is not `nil` safe, and the usual precautions apply.
--
-- Note: all functions taking `i1,j1,i2,j2` as arguments will normalize the
-- arguments using `default_range`.
--
-- Dependencies: `pl.utils`, `pl.tablex`, `pl.types`
-- @module pl.array2d

local tonumber,tostring,io,ipairs,string,table =
    _G.tonumber,_G.tostring,_G.io,_G.ipairs,_G.string,_G.table
local setmetatable,getmetatable = setmetatable,getmetatable

local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local types = require 'pl.types'
local imap,tmap,reduce,keys,tmap2,tset,index_by = tablex.imap,tablex.map,tablex.reduce,tablex.keys,tablex.map2,tablex.set,tablex.index_by
local remove = table.remove
local splitv,fprintf,assert_arg = utils.splitv,utils.fprintf,utils.assert_arg
local byte = string.byte
local stdout = io.stdout
local min = math.min


local array2d = {}

local function obj (int,out)
    local mt = getmetatable(int)
    if mt then
        setmetatable(out,mt)
    end
    return out
end

local function makelist (res)
    return setmetatable(res, require('pl.List'))
end

--- return the row and column size.
-- Size is calculated using the Lua length operator #, so usual precautions
-- regarding `nil` values apply.
-- @array2d a a 2d array
-- @treturn int number of rows (`#a`)
-- @treturn int number of cols (`#a[1]`)
function array2d.size (a)
    assert_arg(1,a,'table')
    return #a,#a[1]
end

do
    local function index (t,k)
        return t[k]
    end

    --- extract a column from the 2D array.
    -- @array2d a 2d array
    -- @param j column index
    -- @return 1d array
    function array2d.column (a,j)
        assert_arg(1,a,'table')
        return makelist(imap(index,a,j))
    end
end
local column = array2d.column

--- extract a row from the 2D array.
-- Added in line with `column`, for read-only purposes directly
-- accessing a[i] is more performant.
-- @array2d a 2d array
-- @param i row index
-- @return 1d array (copy of the row)
function array2d.row(a,i)
    assert_arg(1,a,'table')
    local row = a[i]
    local r = {}
    for n,v in ipairs(row) do
        r[n] = v
    end
    return makelist(r)
end

--- map a function over a 2D array
-- @func f a function of at least one argument
-- @array2d a 2d array
-- @param arg an optional extra argument to be passed to the function.
-- @return 2d array
function array2d.map (f,a,arg)
    assert_arg(2,a,'table')
    f = utils.function_arg(1,f)
    return obj(a,imap(function(row) return imap(f,row,arg) end, a))
end

--- reduce the rows using a function.
-- @func f a binary function
-- @array2d a 2d array
-- @return 1d array
-- @see pl.tablex.reduce
function array2d.reduce_rows (f,a)
    assert_arg(1,a,'table')
    return tmap(function(row) return reduce(f,row) end, a)
end

--- reduce the columns using a function.
-- @func f a binary function
-- @array2d a 2d array
-- @return 1d array
-- @see pl.tablex.reduce
function array2d.reduce_cols (f,a)
    assert_arg(1,a,'table')
    return tmap(function(c) return reduce(f,column(a,c)) end, keys(a[1]))
end

--- reduce a 2D array into a scalar, using two operations.
-- @func opc operation to reduce the final result
-- @func opr operation to reduce the rows
-- @param a 2D array
function array2d.reduce2 (opc,opr,a)
    assert_arg(3,a,'table')
    local tmp = array2d.reduce_rows(opr,a)
    return reduce(opc,tmp)
end

--- map a function over two arrays.
-- They can be both or either 2D arrays
-- @func f function of at least two arguments
-- @int ad order of first array (`1` if `a` is a list/array, `2` if it is a 2d array)
-- @int bd order of second array (`1` if `b` is a list/array, `2` if it is a 2d array)
-- @tab a 1d or 2d array
-- @tab b 1d or 2d array
-- @param arg optional extra argument to pass to function
-- @return 2D array, unless both arrays are 1D
function array2d.map2 (f,ad,bd,a,b,arg)
    assert_arg(1,a,'table')
    assert_arg(2,b,'table')
    f = utils.function_arg(1,f)
    if ad == 1 and bd == 2 then
        return imap(function(row)
            return tmap2(f,a,row,arg)
        end, b)
    elseif ad == 2 and bd == 1 then
        return imap(function(row)
            return tmap2(f,row,b,arg)
        end, a)
    elseif ad == 1 and bd == 1 then
        return tmap2(f,a,b)
    elseif ad == 2 and bd == 2 then
        return tmap2(function(rowa,rowb)
            return tmap2(f,rowa,rowb,arg)
        end, a,b)
    end
end

--- cartesian product of two 1d arrays.
-- @func f a function of 2 arguments
-- @array t1 a 1d table
-- @array t2 a 1d table
-- @return 2d table
-- @usage product('..',{1,2},{'a','b'}) == {{'1a','2a'},{'1b','2b'}}
function array2d.product (f,t1,t2)
    f = utils.function_arg(1,f)
    assert_arg(2,t1,'table')
    assert_arg(3,t2,'table')
    local res = {}
    for i,v in ipairs(t2) do
        res[i] = tmap(f,t1,v)
    end
    return res
end

--- flatten a 2D array.
-- (this goes over columns first.)
-- @array2d t 2d table
-- @return a 1d table
-- @usage flatten {{1,2},{3,4},{5,6}} == {1,2,3,4,5,6}
function array2d.flatten (t)
    local res = {}
    local k = 1
    local rows, cols = array2d.size(t)
    for r = 1, rows do
        local row = t[r]
        for c = 1, cols do
            res[k] = row[c]
            k = k + 1
        end
    end
    return makelist(res)
end

--- reshape a 2D array. Reshape the aray by specifying a new nr of rows.
-- @array2d t 2d array
-- @int nrows new number of rows
-- @bool co use column-order (Fortran-style) (default false)
-- @return a new 2d array
function array2d.reshape (t,nrows,co)
    local nr,nc = array2d.size(t)
    local ncols = nr*nc / nrows
    local res = {}
    local ir,ic = 1,1
    for i = 1,nrows do
        local row = {}
        for j = 1,ncols do
            row[j] = t[ir][ic]
            if not co then
                ic = ic + 1
                if ic > nc then
                    ir = ir + 1
                    ic = 1
                end
            else
                ir = ir + 1
                if ir > nr then
                    ic = ic + 1
                    ir = 1
                end
            end
        end
        res[i] = row
    end
    return obj(t,res)
end

--- transpose a 2D array.
-- @array2d t 2d array
-- @return a new 2d array
function array2d.transpose(t)
  assert_arg(1,t,'table')
  local _, c = array2d.size(t)
  return array2d.reshape(t,c,true)
end

--- swap two rows of an array.
-- @array2d t a 2d array
-- @int i1 a row index
-- @int i2 a row index
-- @return t (same, modified 2d array)
function array2d.swap_rows (t,i1,i2)
    assert_arg(1,t,'table')
    t[i1],t[i2] = t[i2],t[i1]
    return t
end

--- swap two columns of an array.
-- @array2d t a 2d array
-- @int j1 a column index
-- @int j2 a column index
-- @return t (same, modified 2d array)
function array2d.swap_cols (t,j1,j2)
    assert_arg(1,t,'table')
    for _, row in ipairs(t) do
        row[j1],row[j2] = row[j2],row[j1]
    end
    return t
end

--- extract the specified rows.
-- @array2d t 2d array
-- @tparam {int} ridx a table of row indices
-- @return a new 2d array with the extracted rows
function array2d.extract_rows (t,ridx)
    return obj(t,index_by(t,ridx))
end

--- extract the specified columns.
-- @array2d t 2d array
-- @tparam {int} cidx a table of column indices
-- @return a new 2d array with the extracted colums
function array2d.extract_cols (t,cidx)
    assert_arg(1,t,'table')
    local res = {}
    for i = 1,#t do
        res[i] = index_by(t[i],cidx)
    end
    return obj(t,res)
end

--- remove a row from an array.
-- @function array2d.remove_row
-- @array2d t a 2d array
-- @int i a row index
array2d.remove_row = remove

--- remove a column from an array.
-- @array2d t a 2d array
-- @int j a column index
function array2d.remove_col (t,j)
    assert_arg(1,t,'table')
    for i = 1,#t do
        remove(t[i],j)
    end
end

do
    local function _parse (s)
        local r, c = s:match 'R(%d+)C(%d+)'
        if r then
            r,c = tonumber(r),tonumber(c)
            return r,c
        end
        c,r = s:match '(%a+)(%d+)'
        if c then
            local cv = 0
            for i = 1, #c do
              cv = cv * 26 + byte(c:sub(i,i)) - byte 'A' + 1
            end
            return tonumber(r), cv
        end
        error('bad cell specifier: '..s)
    end

    --- parse a spreadsheet range or cell.
    -- The range/cell can be specified either as 'A1:B2' or 'R1C1:R2C2' or for
    -- single cells as 'A1' or 'R1C1'.
    -- @string s a range (case insensitive).
    -- @treturn int start row
    -- @treturn int start col
    -- @treturn int end row (or `nil` if the range was a single cell)
    -- @treturn int end col (or `nil` if the range was a single cell)
    function array2d.parse_range (s)
        assert_arg(1,s,'string')
        s = s:upper()
        if s:find ':' then
            local start,finish = splitv(s,':')
            local i1,j1 = _parse(start)
            local i2,j2 = _parse(finish)
            return i1,j1,i2,j2
        else -- single value
            local i,j = _parse(s)
            return i,j
        end
    end
end

--- get a slice of a 2D array.
-- Same as `slice`.
-- @see slice
function array2d.range (...)
    return array2d.slice(...)
end

local default_range do
    local function norm_value(v, max)
        if not v then return v end
        if v < 0 then
            v = max + v + 1
        end
        if v < 1 then v = 1 end
        if v > max then v = max end
        return v
    end

    --- normalizes coordinates to valid positive entries and defaults.
    -- Negative indices will be counted from the end, too low, or too high
    -- will be limited by the array sizes.
    -- @array2d t a 2D array
    -- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
    -- @tparam[opt=1] int j1 start col
    -- @tparam[opt=N] int i2 end row
    -- @tparam[opt=M] int j2 end col
    -- @see parse_range
    -- @return i1, j1, i2, j2
    function array2d.default_range (t,i1,j1,i2,j2)
        if (type(i1) == 'string') and not (j1 or i2 or j2) then
            i1, j1, i2, j2 = array2d.parse_range(i1)
        end
        local nr, nc = array2d.size(t)
        i1 = norm_value(i1 or 1, nr)
        j1 = norm_value(j1 or 1, nc)
        i2 = norm_value(i2 or nr, nr)
        j2 = norm_value(j2 or nc, nc)
        return i1,j1,i2,j2
    end
    default_range = array2d.default_range
end

--- get a slice of a 2D array. Note that if the specified range has
-- a 1D result, the rank of the result will be 1.
-- @array2d t a 2D array
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
-- @return an array, 2D in general but 1D in special cases.
function array2d.slice (t,i1,j1,i2,j2)
    assert_arg(1,t,'table')
    i1,j1,i2,j2 = default_range(t,i1,j1,i2,j2)
    local res = {}
    for i = i1,i2 do
        local val
        local row = t[i]
        if j1 == j2 then
            val = row[j1]
        else
            val = {}
            for j = j1,j2 do
                val[#val+1] = row[j]
            end
        end
        res[#res+1] = val
    end
    if i1 == i2 then res = res[1] end
    return obj(t,res)
end

--- set a specified range of an array to a value.
-- @array2d t a 2D array
-- @param value the value (may be a function, called as `val(i,j)`)
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
-- @see tablex.set
function array2d.set (t,value,i1,j1,i2,j2)
    i1,j1,i2,j2 = default_range(t,i1,j1,i2,j2)
    local i = i1
    if types.is_callable(value) then
        local old_f = value
        value = function(j)
            return old_f(i,j)
        end
    end
    while i <= i2 do
        tset(t[i],value,j1,j2)
        i = i + 1
    end
end

--- write a 2D array to a file.
-- @array2d t a 2D array
-- @param f a file object (default stdout)
-- @string fmt a format string (default is just to use tostring)
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
function array2d.write (t,f,fmt,i1,j1,i2,j2)
    assert_arg(1,t,'table')
    f = f or stdout
    local rowop
    if fmt then
        rowop = function(row,j) fprintf(f,fmt,row[j]) end
    else
        rowop = function(row,j) f:write(tostring(row[j]),' ') end
    end
    local function newline()
        f:write '\n'
    end
    array2d.forall(t,rowop,newline,i1,j1,i2,j2)
end

--- perform an operation for all values in a 2D array.
-- @array2d t 2D array
-- @func row_op function to call on each value; `row_op(row,j)`
-- @func end_row_op function to call at end of each row; `end_row_op(i)`
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
function array2d.forall (t,row_op,end_row_op,i1,j1,i2,j2)
    assert_arg(1,t,'table')
    i1,j1,i2,j2 = default_range(t,i1,j1,i2,j2)
    for i = i1,i2 do
        local row = t[i]
        for j = j1,j2 do
            row_op(row,j)
        end
        if end_row_op then end_row_op(i) end
    end
end

---- move a block from the destination to the source.
-- @array2d dest a 2D array
-- @int di start row in dest
-- @int dj start col in dest
-- @array2d src a 2D array
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
function array2d.move (dest,di,dj,src,i1,j1,i2,j2)
    assert_arg(1,dest,'table')
    assert_arg(4,src,'table')
    i1,j1,i2,j2 = default_range(src,i1,j1,i2,j2)
    local nr,nc = array2d.size(dest)
    i2, j2 = min(nr,i2), min(nc,j2)
    --i1, j1 = max(1,i1), max(1,j1)
    dj = dj - 1
    for i = i1,i2 do
        local drow, srow = dest[i+di-1], src[i]
        for j = j1,j2 do
            drow[j+dj] = srow[j]
        end
    end
end

--- iterate over all elements in a 2D array, with optional indices.
-- @array2d a 2D array
-- @bool indices with indices (default false)
-- @tparam[opt=1] int|string i1 start row or spreadsheet range passed to `parse_range`
-- @tparam[opt=1] int j1 start col
-- @tparam[opt=N] int i2 end row
-- @tparam[opt=M] int j2 end col
-- @see parse_range
-- @return either `value` or `i,j,value` depending on the value of `indices`
function array2d.iter(a,indices,i1,j1,i2,j2)
    assert_arg(1,a,'table')
    i1,j1,i2,j2 = default_range(a,i1,j1,i2,j2)
    local i,j = i1,j1-1
    local row = a[i]
    return function()
        j = j + 1
        if j > j2 then
            j = j1
            i = i + 1
            row = a[i]
            if i > i2 then
                return nil
            end
        end
        if indices then
            return i,j,row[j]
        else
            return row[j]
        end
    end
end

--- iterate over all columns.
-- @array2d a a 2D array
-- @return column, column-index
function array2d.columns(a)
  assert_arg(1,a,'table')
  local n = #a[1]
  local i = 0
  return function()
      i = i + 1
      if i > n then return nil end
      return column(a,i), i
  end
end

--- iterate over all rows.
-- Returns a copy of the row, for read-only purposes directly iterating
-- is more performant; `ipairs(a)`
-- @array2d a a 2D array
-- @return row, row-index
function array2d.rows(a)
  assert_arg(1,a,'table')
  local n = #a
  local i = 0
  return function()
      i = i + 1
      if i > n then return nil end
      return array2d.row(a,i), i
  end
end

--- new array of specified dimensions
-- @int rows number of rows
-- @int cols number of cols
-- @param val initial value; if it's a function then use `val(i,j)`
-- @return new 2d array
function array2d.new(rows,cols,val)
    local res = {}
    local fun = types.is_callable(val)
    for i = 1,rows do
        local row = {}
        if fun then
            for j = 1,cols do row[j] = val(i,j) end
        else
            for j = 1,cols do row[j] = val end
        end
        res[i] = row
    end
    return res
end

return array2d
