--- Permutation operations.
--
-- Dependencies: `pl.utils`, `pl.tablex`
-- @module pl.permute
local tablex = require 'pl.tablex'
local utils = require 'pl.utils'
local copy = tablex.deepcopy
local append = table.insert
local assert_arg = utils.assert_arg


local permute = {}


--- an iterator over all order-permutations of the elements of a list.
-- Please note that the same list is returned each time, so do not keep references!
-- @param a list-like table
-- @return an iterator which provides the next permutation as a list
function permute.order_iter(a)
    assert_arg(1,a,'table')

    local t = #a
    local stack = { 1 }
    local function iter()
        local h = #stack
        local n = t - h + 1

        local i = stack[h]
        if i > t then
            return
        end

        if n == 0 then
            table.remove(stack)
            h = h - 1

            stack[h] = stack[h] + 1
            return a

        elseif i <= n then

            -- put i-th element as the last one
            a[n], a[i] = a[i], a[n]

            -- generate all permutations of the other elements
            table.insert(stack, 1)

        else

            table.remove(stack)
            h = h - 1

            n = n + 1
            i = stack[h]

            -- restore i-th element
            a[n], a[i] = a[i], a[n]

            stack[h] = stack[h] + 1
        end
        return iter() -- tail-call
    end

    return iter
end


--- construct a table containing all the order-permutations of a list.
-- @param a list-like table
-- @return a table of tables
-- @usage permute.order_table {1,2,3} --> {{2,3,1},{3,2,1},{3,1,2},{1,3,2},{2,1,3},{1,2,3}}
function permute.order_table (a)
    assert_arg(1,a,'table')
    local res = {}
    for t in permute.iter(a) do
        append(res,copy(t))
    end
    return res
end



--- an iterator over all permutations of the elements of the given lists.
-- @param ... list-like tables, they are nil-safe if a length-field `n` is provided (see `utils.pack`)
-- @return an iterator which provides the next permutation as return values in the same order as the provided lists, preceeded by an index
-- @usage
-- local strs = utils.pack("one", nil, "three")  -- adds an 'n' field for nil-safety
-- local bools = utils.pack(true, false)
-- local iter = permute.list_iter(strs, bools)
--
-- print(iter())    --> 1, one, true
-- print(iter())    --> 2, nil, true
-- print(iter())    --> 3, three, true
-- print(iter())    --> 4, one, false
-- print(iter())    --> 5, nil, false
-- print(iter())    --> 6, three, false
function permute.list_iter(...)
  local elements = {...}
  local pointers = {}
  local sizes = {}
  local size = #elements
  for i, list in ipairs(elements) do
    assert_arg(i,list,'table')
    pointers[i] = 1
    sizes[i] = list.n or #list
  end
  local count = 0

  return function()
    if pointers[size] > sizes[size] then return end -- we're done
    count = count + 1
    local r = { n = #elements }
    local cascade_up = true
    for i = 1, size do
      r[i] = elements[i][pointers[i]]
      if cascade_up then
        pointers[i] = pointers[i] + 1
        if pointers[i] <= sizes[i] then
          -- this list is not done yet, stop cascade
          cascade_up = false
        else
          -- this list is done
          if i ~= size then
            -- reset pointer
            pointers[i] = 1
          end
        end
      end
    end
    return count, utils.unpack(r)
  end
end



--- construct a table containing all the permutations of a set of lists.
-- @param ... list-like tables, they are nil-safe if a length-field `n` is provided
-- @return a list of lists, the sub-lists have an 'n' field for nil-safety
-- @usage
-- local strs = utils.pack("one", nil, "three")  -- adds an 'n' field for nil-safety
-- local bools = utils.pack(true, false)
-- local results = permute.list_table(strs, bools)
-- -- results = {
-- --   { "one, true, n = 2 }
-- --   { nil, true, n = 2 },
-- --   { "three, true, n = 2 },
-- --   { "one, false, n = 2 },
-- --   { nil, false, n = 2 },
-- --   { "three", false, n = 2 },
-- -- }
function permute.list_table(...)
  local iter = permute.list_iter(...)
  local results = {}
  local i = 1
  while true do
    local values = utils.pack(iter())
    if values[1] == nil then return results end
    for i = 1, values.n do values[i] = values[i+1] end
    values.n = values.n - 1
    results[i] = values
    i = i + 1
  end
end


-- backward compat, to be deprecated

--- deprecated.
-- @param ...
-- @see permute.order_iter
function permute.iter(...)
  utils.raise_deprecation {
    source = "Penlight " .. utils._VERSION,
    message = "function 'iter' was renamed to 'order_iter'",
    version_removed = "2.0.0",
    deprecated_after = "1.9.2",
  }

  return permute.order_iter(...)
end

--- deprecated.
-- @param ...
-- @see permute.order_iter
function permute.table(...)
  utils.raise_deprecation {
    source = "Penlight " .. utils._VERSION,
    message = "function 'table' was renamed to 'order_table'",
    version_removed = "2.0.0",
    deprecated_after = "1.9.2",
  }

  return permute.order_table(...)
end

return permute
