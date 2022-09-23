--- Python-style extended string library.
--
-- see 3.6.1 of the Python reference.
-- If you want to make these available as string methods, then say
-- `stringx.import()` to bring them into the standard `string` table.
--
-- See @{03-strings.md|the Guide}
--
-- Dependencies: `pl.utils`, `pl.types`
-- @module pl.stringx
local utils = require 'pl.utils'
local is_callable = require 'pl.types'.is_callable
local string = string
local find = string.find
local type,setmetatable,ipairs = type,setmetatable,ipairs
local error = error
local gsub = string.gsub
local rep = string.rep
local sub = string.sub
local reverse = string.reverse
local concat = table.concat
local append = table.insert
local remove = table.remove
local escape = utils.escape
local ceil, max = math.ceil, math.max
local assert_arg,usplit = utils.assert_arg,utils.split
local lstrip
local unpack = utils.unpack
local pack = utils.pack

local function assert_string (n,s)
    assert_arg(n,s,'string')
end

local function non_empty(s)
    return #s > 0
end

local function assert_nonempty_string(n,s)
    assert_arg(n,s,'string',non_empty,'must be a non-empty string')
end

local function makelist(l)
    return setmetatable(l, require('pl.List'))
end

local stringx = {}

------------------
-- String Predicates
-- @section predicates

--- does s only contain alphabetic characters?
-- @string s a string
function stringx.isalpha(s)
    assert_string(1,s)
    return find(s,'^%a+$') == 1
end

--- does s only contain digits?
-- @string s a string
function stringx.isdigit(s)
    assert_string(1,s)
    return find(s,'^%d+$') == 1
end

--- does s only contain alphanumeric characters?
-- @string s a string
function stringx.isalnum(s)
    assert_string(1,s)
    return find(s,'^%w+$') == 1
end

--- does s only contain whitespace?
-- Matches on pattern '%s' so matches space, newline, tabs, etc.
-- @string s a string
function stringx.isspace(s)
    assert_string(1,s)
    return find(s,'^%s+$') == 1
end

--- does s only contain lower case characters?
-- @string s a string
function stringx.islower(s)
    assert_string(1,s)
    return find(s,'^[%l%s]+$') == 1
end

--- does s only contain upper case characters?
-- @string s a string
function stringx.isupper(s)
    assert_string(1,s)
    return find(s,'^[%u%s]+$') == 1
end

local function raw_startswith(s, prefix)
    return find(s,prefix,1,true) == 1
end

local function raw_endswith(s, suffix)
    return #s >= #suffix and find(s, suffix, #s-#suffix+1, true) and true or false
end

local function test_affixes(s, affixes, fn)
    if type(affixes) == 'string' then
        return fn(s,affixes)
    elseif type(affixes) == 'table' then
        for _,affix in ipairs(affixes) do
            if fn(s,affix) then return true end
        end
        return false
    else
        error(("argument #2 expected a 'string' or a 'table', got a '%s'"):format(type(affixes)))
    end
end

--- does s start with prefix or one of prefixes?
-- @string s a string
-- @param prefix a string or an array of strings
function stringx.startswith(s,prefix)
    assert_string(1,s)
    return test_affixes(s,prefix,raw_startswith)
end

--- does s end with suffix or one of suffixes?
-- @string s a string
-- @param suffix a string or an array of strings
function stringx.endswith(s,suffix)
    assert_string(1,s)
    return test_affixes(s,suffix,raw_endswith)
end

--- Strings and Lists
-- @section lists

--- concatenate the strings using this string as a delimiter.
-- Note that the arguments are reversed from `string.concat`.
-- @string s the string
-- @param seq a table of strings or numbers
-- @usage stringx.join(' ', {1,2,3}) == '1 2 3'
function stringx.join(s,seq)
    assert_string(1,s)
    return concat(seq,s)
end

--- Split a string into a list of lines.
-- `"\r"`, `"\n"`, and `"\r\n"` are considered line ends.
-- They are not included in the lines unless `keepends` is passed.
-- Terminal line end does not produce an extra line.
-- Splitting an empty string results in an empty list.
-- @string s the string.
-- @bool[opt] keep_ends include line ends.
-- @return List of lines
function stringx.splitlines(s, keep_ends)
    assert_string(1, s)
    local res = {}
    local pos = 1
    while true do
        local line_end_pos = find(s, '[\r\n]', pos)
        if not line_end_pos then
            break
        end

        local line_end = sub(s, line_end_pos, line_end_pos)
        if line_end == '\r' and sub(s, line_end_pos + 1, line_end_pos + 1) == '\n' then
            line_end = '\r\n'
        end

        local line = sub(s, pos, line_end_pos - 1)
        if keep_ends then
            line = line .. line_end
        end
        append(res, line)

        pos = line_end_pos + #line_end
    end

    if pos <= #s then
        append(res, sub(s, pos))
    end
    return makelist(res)
end

--- split a string into a list of strings using a delimiter.
-- @function split
-- @string s the string
-- @string[opt] re a delimiter (defaults to whitespace)
-- @int[opt] n maximum number of results
-- @return List
-- @usage #(stringx.split('one two')) == 2
-- @usage stringx.split('one,two,three', ',') == List{'one','two','three'}
-- @usage stringx.split('one,two,three', ',', 2) == List{'one','two,three'}
function stringx.split(s,re,n)
    assert_string(1,s)
    local plain = true
    if not re then -- default spaces
        s = lstrip(s)
        plain = false
    end
    local res = usplit(s,re,plain,n)
    if re and re ~= '' and
       find(s,re,-#re,true) and
       (n or math.huge) > #res then
        res[#res+1] = ""
    end
    return makelist(res)
end

--- replace all tabs in s with tabsize spaces. If not specified, tabsize defaults to 8.
-- Tab stops will be honored.
-- @string s the string
-- @int tabsize[opt=8] number of spaces to expand each tab
-- @return expanded string
-- @usage stringx.expandtabs('\tone,two,three', 4)   == '    one,two,three'
-- @usage stringx.expandtabs('  \tone,two,three', 4) == '    one,two,three'
function stringx.expandtabs(s,tabsize)
  assert_string(1,s)
  tabsize = tabsize or 8
  return (s:gsub("([^\t\r\n]*)\t", function(before_tab)
      if tabsize == 0 then
        return before_tab
      else
        return before_tab .. (" "):rep(tabsize - #before_tab % tabsize)
      end
    end))
end

--- Finding and Replacing
-- @section find

local function _find_all(s,sub,first,last,allow_overlap)
    first = first or 1
    last = last or #s
    if sub == '' then return last+1,last-first+1 end
    local i1,i2 = find(s,sub,first,true)
    local res
    local k = 0
    while i1 do
        if last and i2 > last then break end
        res = i1
        k = k + 1
        if allow_overlap then
            i1,i2 = find(s,sub,i1+1,true)
        else
            i1,i2 = find(s,sub,i2+1,true)
        end
    end
    return res,k
end

--- find index of first instance of sub in s from the left.
-- @string s the string
-- @string sub substring
-- @int[opt] first first index
-- @int[opt] last last index
-- @return start index, or nil if not found
function stringx.lfind(s,sub,first,last)
    assert_string(1,s)
    assert_string(2,sub)
    local i1, i2 = find(s,sub,first,true)

    if i1 and (not last or i2 <= last) then
        return i1
    else
        return nil
    end
end

--- find index of first instance of sub in s from the right.
-- @string s the string
-- @string sub substring
-- @int[opt] first first index
-- @int[opt] last last index
-- @return start index, or nil if not found
function stringx.rfind(s,sub,first,last)
    assert_string(1,s)
    assert_string(2,sub)
    return (_find_all(s,sub,first,last,true))
end

--- replace up to n instances of old by new in the string s.
-- If n is not present, replace all instances.
-- @string s the string
-- @string old the target substring
-- @string new the substitution
-- @int[opt] n optional maximum number of substitutions
-- @return result string
function stringx.replace(s,old,new,n)
    assert_string(1,s)
    assert_string(2,old)
    assert_string(3,new)
    return (gsub(s,escape(old),new:gsub('%%','%%%%'),n))
end

--- count all instances of substring in string.
-- @string s the string
-- @string sub substring
-- @bool[opt] allow_overlap allow matches to overlap
-- @usage
-- assert(stringx.count('banana', 'ana') == 1)
-- assert(stringx.count('banana', 'ana', true) == 2)
function stringx.count(s,sub,allow_overlap)
    assert_string(1,s)
    local _,k = _find_all(s,sub,1,false,allow_overlap)
    return k
end

--- Stripping and Justifying
-- @section strip

local function _just(s,w,ch,left,right)
    local n = #s
    if w > n then
        if not ch then ch = ' ' end
        local f1,f2
        if left and right then
            local rn = ceil((w-n)/2)
            local ln = w - n - rn
            f1 = rep(ch,ln)
            f2 = rep(ch,rn)
        elseif right then
            f1 = rep(ch,w-n)
            f2 = ''
        else
            f2 = rep(ch,w-n)
            f1 = ''
        end
        return f1..s..f2
    else
        return s
    end
end

--- left-justify s with width w.
-- @string s the string
-- @int w width of justification
-- @string[opt=' '] ch padding character
-- @usage stringx.ljust('hello', 10, '*') == '*****hello'
function stringx.ljust(s,w,ch)
    assert_string(1,s)
    assert_arg(2,w,'number')
    return _just(s,w,ch,true,false)
end

--- right-justify s with width w.
-- @string s the string
-- @int w width of justification
-- @string[opt=' '] ch padding character
-- @usage stringx.rjust('hello', 10, '*') == 'hello*****'
function stringx.rjust(s,w,ch)
    assert_string(1,s)
    assert_arg(2,w,'number')
    return _just(s,w,ch,false,true)
end

--- center-justify s with width w.
-- @string s the string
-- @int w width of justification
-- @string[opt=' '] ch padding character
-- @usage stringx.center('hello', 10, '*') == '**hello***'
function stringx.center(s,w,ch)
    assert_string(1,s)
    assert_arg(2,w,'number')
    return _just(s,w,ch,true,true)
end

local function _strip(s,left,right,chrs)
    if not chrs then
        chrs = '%s'
    else
        chrs = '['..escape(chrs)..']'
    end
    local f = 1
    local t
    if left then
        local i1,i2 = find(s,'^'..chrs..'*')
        if i2 >= i1 then
            f = i2+1
        end
    end
    if right then
        if #s < 200 then
            local i1,i2 = find(s,chrs..'*$',f)
            if i2 >= i1 then
                t = i1-1
            end
        else
            local rs = reverse(s)
            local i1,i2 = find(rs, '^'..chrs..'*')
            if i2 >= i1 then
                t = -i2-1
            end
        end
    end
    return sub(s,f,t)
end

--- trim any characters on the left of s.
-- @string s the string
-- @string[opt='%s'] chrs default any whitespace character,
-- but can be a string of characters to be trimmed
function stringx.lstrip(s,chrs)
    assert_string(1,s)
    return _strip(s,true,false,chrs)
end
lstrip = stringx.lstrip

--- trim any characters on the right of s.
-- @string s the string
-- @string[opt='%s'] chrs default any whitespace character,
-- but can be a string of characters to be trimmed
function stringx.rstrip(s,chrs)
    assert_string(1,s)
    return _strip(s,false,true,chrs)
end

--- trim any characters on both left and right of s.
-- @string s the string
-- @string[opt='%s'] chrs default any whitespace character,
-- but can be a string of characters to be trimmed
-- @usage stringx.strip('  --== Hello ==--  ', "- =")  --> 'Hello'
function stringx.strip(s,chrs)
    assert_string(1,s)
    return _strip(s,true,true,chrs)
end

--- Partitioning Strings
-- @section partitioning

--- split a string using a pattern. Note that at least one value will be returned!
-- @string s the string
-- @string[opt='%s'] re a Lua string pattern (defaults to whitespace)
-- @return the parts of the string
-- @usage  a,b = line:splitv('=')
-- @see utils.splitv
function stringx.splitv(s,re)
    assert_string(1,s)
    return utils.splitv(s,re)
end

-- The partition functions split a string using a delimiter into three parts:
-- the part before, the delimiter itself, and the part afterwards
local function _partition(p,delim,fn)
    local i1,i2 = fn(p,delim)
    if not i1 or i1 == -1 then
        return p,'',''
    else
        if not i2 then i2 = i1 end
        return sub(p,1,i1-1),sub(p,i1,i2),sub(p,i2+1)
    end
end

--- partition the string using first occurance of a delimiter
-- @string s the string
-- @string ch delimiter (match as plain string, no patterns)
-- @return part before ch
-- @return ch
-- @return part after ch
-- @usage {stringx.partition('a,b,c', ','))} == {'a', ',', 'b,c'}
-- @usage {stringx.partition('abc', 'x'))} == {'abc', '', ''}
function stringx.partition(s,ch)
    assert_string(1,s)
    assert_nonempty_string(2,ch)
    return _partition(s,ch,stringx.lfind)
end

--- partition the string p using last occurance of a delimiter
-- @string s the string
-- @string ch delimiter (match as plain string, no patterns)
-- @return part before ch
-- @return ch
-- @return part after ch
-- @usage {stringx.rpartition('a,b,c', ','))} == {'a,b', ',', 'c'}
-- @usage {stringx.rpartition('abc', 'x'))} == {'', '', 'abc'}
function stringx.rpartition(s,ch)
    assert_string(1,s)
    assert_nonempty_string(2,ch)
    local a,b,c = _partition(s,ch,stringx.rfind)
    if a == s then -- no match found
        return c,b,a
    end
    return a,b,c
end

--- return the 'character' at the index.
-- @string s the string
-- @int idx an index (can be negative)
-- @return a substring of length 1 if successful, empty string otherwise.
function stringx.at(s,idx)
    assert_string(1,s)
    assert_arg(2,idx,'number')
    return sub(s,idx,idx)
end


--- Text handling
-- @section text


--- indent a multiline string.
-- @tparam string s the (multiline) string
-- @tparam integer n the size of the indent
-- @tparam[opt=' '] string ch the character to use when indenting
-- @return indented string
function stringx.indent (s,n,ch)
  assert_arg(1,s,'string')
  assert_arg(2,n,'number')
  local lines = usplit(s ,'\n')
  local prefix = string.rep(ch or ' ',n)
  for i, line in ipairs(lines) do
    lines[i] = prefix..line
  end
  return concat(lines,'\n')..'\n'
end


--- dedent a multiline string by removing any initial indent.
-- useful when working with [[..]] strings.
-- Empty lines are ignored.
-- @tparam string s the (multiline) string
-- @return a string with initial indent zero.
-- @usage
-- local s = dedent [[
--          One
--
--        Two
--
--      Three
-- ]]
-- assert(s == [[
--     One
--
--   Two
--
-- Three
-- ]])
function stringx.dedent (s)
  assert_arg(1,s,'string')
  local lst = usplit(s,'\n')
  if #lst>0 then
    local ind_size = math.huge
    for i, line in ipairs(lst) do
      local i1, i2 = lst[i]:find('^%s*[^%s]')
      if i1 and i2 < ind_size then
        ind_size = i2
      end
    end
    for i, line in ipairs(lst) do
      lst[i] = lst[i]:sub(ind_size, -1)
    end
  end
  return concat(lst,'\n')..'\n'
end



do
  local buildline = function(words, size, breaklong)
    -- if overflow is set, a word longer than size, will overflow the size
    -- otherwise it will be chopped in line-length pieces
    local line = {}
    if #words[1] > size then
      -- word longer than line
      if not breaklong then
        line[1] = words[1]
        remove(words, 1)
      else
        line[1] = words[1]:sub(1, size)
        words[1] = words[1]:sub(size + 1, -1)
      end
    else
      local len = 0
      while words[1] and (len + #words[1] <= size) or
            (len == 0 and #words[1] == size) do
        if words[1] ~= "" then
          line[#line+1] = words[1]
          len = len + #words[1] + 1
        end
        remove(words, 1)
      end
    end
    return stringx.strip(concat(line, " ")), words
  end

  --- format a paragraph into lines so that they fit into a line width.
  -- It will not break long words by default, so lines can be over the length
  -- to that extent.
  -- @tparam string s the string to format
  -- @tparam[opt=70] integer width the margin width
  -- @tparam[opt=false] boolean breaklong if truthy, words longer than the width given will be forced split.
  -- @return a list of lines (List object), use `fill` to return a string instead of a `List`.
  -- @see pl.List
  -- @see fill
  stringx.wrap = function(s, width, breaklong)
    s = s:gsub('\n',' ') -- remove line breaks
    s = stringx.strip(s) -- remove leading/trailing whitespace
    if s == "" then
      return { "" }
    end
    width = width or 70
    local out = {}
    local words = usplit(s, "%s")
    while words[1] do
      out[#out+1], words = buildline(words, width, breaklong)
    end
    return makelist(out)
  end
end

--- format a paragraph so that it fits into a line width.
-- @tparam string s the string to format
-- @tparam[opt=70] integer width the margin width
-- @tparam[opt=false] boolean breaklong if truthy, words longer than the width given will be forced split.
-- @return a string, use `wrap` to return a list of lines instead of a string.
-- @see wrap
function stringx.fill (s,width,breaklong)
  return concat(stringx.wrap(s,width,breaklong),'\n') .. '\n'
end

--- Template
-- @section Template


local function _substitute(s,tbl,safe)
  local subst
  if is_callable(tbl) then
    subst = tbl
  else
    function subst(f)
      local s = tbl[f]
      if not s then
        if safe then
          return f
        else
          error("not present in table "..f)
        end
      else
        return s
      end
    end
  end
  local res = gsub(s,'%${([%w_]+)}',subst)
  return (gsub(res,'%$([%w_]+)',subst))
end



local Template = {}
stringx.Template = Template
Template.__index = Template
setmetatable(Template, {
  __call = function(obj,tmpl)
    return Template.new(tmpl)
  end
})

--- Creates a new Template class.
-- This is a shortcut to `Template.new(tmpl)`.
-- @tparam string tmpl the template string
-- @function Template
-- @treturn Template
function Template.new(tmpl)
  assert_arg(1,tmpl,'string')
  local res = {}
  res.tmpl = tmpl
  setmetatable(res,Template)
  return res
end

--- substitute values into a template, throwing an error.
-- This will throw an error if no name is found.
-- @tparam table tbl a table of name-value pairs.
-- @return string with place holders substituted
function Template:substitute(tbl)
  assert_arg(1,tbl,'table')
  return _substitute(self.tmpl,tbl,false)
end

--- substitute values into a template.
-- This version just passes unknown names through.
-- @tparam table tbl a table of name-value pairs.
-- @return string with place holders substituted
function Template:safe_substitute(tbl)
  assert_arg(1,tbl,'table')
  return _substitute(self.tmpl,tbl,true)
end

--- substitute values into a template, preserving indentation. <br>
-- If the value is a multiline string _or_ a template, it will insert
-- the lines at the correct indentation. <br>
-- Furthermore, if a template, then that template will be substituted
-- using the same table.
-- @tparam table tbl a table of name-value pairs.
-- @return string with place holders substituted
function Template:indent_substitute(tbl)
  assert_arg(1,tbl,'table')
  if not self.strings then
    self.strings = usplit(self.tmpl,'\n')
  end

  -- the idea is to substitute line by line, grabbing any spaces as
  -- well as the $var. If the value to be substituted contains newlines,
  -- then we split that into lines and adjust the indent before inserting.
  local function subst(line)
    return line:gsub('(%s*)%$([%w_]+)',function(sp,f)
      local subtmpl
      local s = tbl[f]
      if not s then error("not present in table "..f) end
      if getmetatable(s) == Template then
        subtmpl = s
        s = s.tmpl
      else
        s = tostring(s)
      end
      if s:find '\n' then
        local lines = usplit(s, '\n')
        for i, line in ipairs(lines) do
          lines[i] = sp..line
        end
        s = concat(lines, '\n') .. '\n'
      end
      if subtmpl then
        return _substitute(s, tbl)
      else
        return s
      end
    end)
  end

  local lines = {}
  for i, line in ipairs(self.strings) do
    lines[i] = subst(line)
  end
  return concat(lines,'\n')..'\n'
end



--- Miscelaneous
-- @section misc

--- return an iterator over all lines in a string
-- @string s the string
-- @return an iterator
-- @usage
-- local line_no = 1
-- for line in stringx.lines(some_text) do
--   print(line_no, line)
--   line_no = line_no + 1
-- end
function stringx.lines(s)
    assert_string(1,s)
    if not s:find '\n$' then s = s..'\n' end
    return s:gmatch('([^\n]*)\n')
end

--- inital word letters uppercase ('title case').
-- Here 'words' mean chunks of non-space characters.
-- @string s the string
-- @return a string with each word's first letter uppercase
-- @usage stringx.title("hello world") == "Hello World")
function stringx.title(s)
    assert_string(1,s)
    return (s:gsub('(%S)(%S*)',function(f,r)
        return f:upper()..r:lower()
    end))
end

stringx.capitalize = stringx.title

do
  local ellipsis = '...'
  local n_ellipsis = #ellipsis

  --- Return a shortened version of a string.
  -- Fits string within w characters. Removed characters are marked with ellipsis.
  -- @string s the string
  -- @int w the maxinum size allowed
  -- @bool tail true if we want to show the end of the string (head otherwise)
  -- @usage ('1234567890'):shorten(8) == '12345...'
  -- @usage ('1234567890'):shorten(8, true) == '...67890'
  -- @usage ('1234567890'):shorten(20) == '1234567890'
  function stringx.shorten(s,w,tail)
      assert_string(1,s)
      if #s > w then
          if w < n_ellipsis then return ellipsis:sub(1,w) end
          if tail then
              local i = #s - w + 1 + n_ellipsis
              return ellipsis .. s:sub(i)
          else
              return s:sub(1,w-n_ellipsis) .. ellipsis
          end
      end
      return s
  end
end


do
  -- Utility function that finds any patterns that match a long string's an open or close.
  -- Note that having this function use the least number of equal signs that is possible is a harder algorithm to come up with.
  -- Right now, it simply returns the greatest number of them found.
  -- @param s The string
  -- @return 'nil' if not found. If found, the maximum number of equal signs found within all matches.
  local function has_lquote(s)
      local lstring_pat = '([%[%]])(=*)%1'
      local equals, new_equals, _
      local finish = 1
      repeat
          _, finish, _, new_equals = s:find(lstring_pat, finish)
          if new_equals then
              equals = max(equals or 0, #new_equals)
          end
      until not new_equals

      return equals
  end

  --- Quote the given string and preserve any control or escape characters, such that reloading the string in Lua returns the same result.
  -- @param s The string to be quoted.
  -- @return The quoted string.
  function stringx.quote_string(s)
      assert_string(1,s)
      -- Find out if there are any embedded long-quote sequences that may cause issues.
      -- This is important when strings are embedded within strings, like when serializing.
      -- Append a closing bracket to catch unfinished long-quote sequences at the end of the string.
      local equal_signs = has_lquote(s .. "]")

      -- Note that strings containing "\r" can't be quoted using long brackets
      -- as Lua lexer converts all newlines to "\n" within long strings.
      if (s:find("\n") or equal_signs) and not s:find("\r") then
          -- If there is an embedded sequence that matches a long quote, then
          -- find the one with the maximum number of = signs and add one to that number.
          equal_signs = ("="):rep((equal_signs or -1) + 1)
          -- Long strings strip out leading newline. We want to retain that, when quoting.
          if s:find("^\n") then s = "\n" .. s end
          local lbracket, rbracket =
              "[" .. equal_signs .. "[",
              "]" .. equal_signs .. "]"
          s = lbracket .. s .. rbracket
      else
          -- Escape funny stuff. Lua 5.1 does not handle "\r" correctly.
          s = ("%q"):format(s):gsub("\r", "\\r")
      end
      return s
  end
end


--- Python-style formatting operator.
-- Calling `text.format_operator()` overloads the % operator for strings to give
-- Python/Ruby style formated output.
-- This is extended to also do template-like substitution for map-like data.
--
-- Note this goes further than the original, and will allow these cases:
--
-- 1. a single value
-- 2. a list of values
-- 3. a map of var=value pairs
-- 4. a function, as in gsub
--
-- For the second two cases, it uses $-variable substituion.
--
-- When called, this function will monkey-patch the global `string` metatable by
-- adding a `__mod` method.
--
-- See <a href="http://lua-users.org/wiki/StringInterpolation">the lua-users wiki</a>
--
-- @usage
-- require 'pl.text'.format_operator()
-- local out1 = '%s = %5.3f' % {'PI',math.pi}                   --> 'PI = 3.142'
-- local out2 = '$name = $value' % {name='dog',value='Pluto'}   --> 'dog = Pluto'
function stringx.format_operator()

  local format = string.format

  -- a more forgiving version of string.format, which applies
  -- tostring() to any value with a %s format.
  local function formatx (fmt,...)
    local args = pack(...)
    local i = 1
    for p in fmt:gmatch('%%.') do
      if p == '%s' and type(args[i]) ~= 'string' then
        args[i] = tostring(args[i])
      end
      i = i + 1
    end
    return format(fmt,unpack(args))
  end

  local function basic_subst(s,t)
    return (s:gsub('%$([%w_]+)',t))
  end

  getmetatable("").__mod = function(a, b)
    if b == nil then
      return a
    elseif type(b) == "table" and getmetatable(b) == nil then
      if #b == 0 then -- assume a map-like table
        return _substitute(a,b,true)
      else
        return formatx(a,unpack(b))
      end
    elseif type(b) == 'function' then
      return basic_subst(a,b)
    else
      return formatx(a,b)
    end
  end
end

--- import the stringx functions into the global string (meta)table
function stringx.import()
    utils.import(stringx,string)
end

return stringx
