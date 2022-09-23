local template = require 'pl.template'
local subst = template.substitute
local List = require 'pl.List'
local asserteq = require 'pl.test'.asserteq
local utils = require 'pl.utils'



asserteq(subst([[
# for i = 1,2 do
<p>Hello $(tostring(i))</p>
# end
]],_G),[[
<p>Hello 1</p>
<p>Hello 2</p>
]])



asserteq(subst([[
<ul>
# for name in ls:iter() do
   <li>$(name)</li>
#end
</ul>
]],{ls = List{'john','alice','jane'}}),[[
<ul>
   <li>john</li>
   <li>alice</li>
   <li>jane</li>
</ul>
]])



-- can change the default escape from '#' so we can do C/C++ output.
-- note that the environment can have a parent field.
asserteq(subst([[
> for i,v in ipairs{'alpha','beta','gamma'} do
    cout << obj.${v} << endl;
> end
]],{_parent=_G, _brackets='{}', _escape='>'}),[[
    cout << obj.alpha << endl;
    cout << obj.beta << endl;
    cout << obj.gamma << endl;
]])



-- handle templates with a lot of substitutions
asserteq(subst(("$(x)\n"):rep(300), {x = "y"}), ("y\n"):rep(300))



--------------------------------------------------
-- Test using no leading nor trailing linebreak
local tmpl = [[<ul>
# for i,val in ipairs(T) do
<li>$(i) = $(val:upper())</li>
# end
</ul>]]

local my_env = {
  ipairs = ipairs,
  T = {'one','two','three'},
  _debug = true,
}
local res, err = template.substitute(tmpl, my_env)

--print(res, err)
asserteq(res, [[<ul>
<li>1 = ONE</li>
<li>2 = TWO</li>
<li>3 = THREE</li>
</ul>]])



--------------------------------------------------
-- Test using both leading and trailing linebreak
local tmpl = [[
<ul>
# for i,val in ipairs(T) do
<li>$(i) = $(val:upper())</li>
# end
</ul>
]]

local my_env = {
  ipairs = ipairs,
  T = {'one','two','three'},
  _debug = true,
}
local res, err = template.substitute(tmpl, my_env)

--print(res, err)
asserteq(res, [[
<ul>
<li>1 = ONE</li>
<li>2 = TWO</li>
<li>3 = THREE</li>
</ul>
]])



--------------------------------------------------
-- Test reusing a compiled template
local tmpl = [[
<ul>
# for i,val in ipairs(T) do
<li>$(i) = $(val:upper())</li>
# end
</ul>
]]

local my_env = {
  ipairs = ipairs,
  T = {'one','two','three'}
}
local t, err = template.compile(tmpl, { debug = true })
local res, err, code = t:render(my_env)
--print(res, err, code)
asserteq(res, [[
<ul>
<li>1 = ONE</li>
<li>2 = TWO</li>
<li>3 = THREE</li>
</ul>
]])


-- now reuse with different env
local my_env = {
  ipairs = ipairs,
  T = {'four','five','six'}
}
local t, err = template.compile(tmpl, { debug = true })
local res, err, code = t:render(my_env)
--print(res, err, code)
asserteq(res, [[
<ul>
<li>1 = FOUR</li>
<li>2 = FIVE</li>
<li>3 = SIX</li>
</ul>
]])



--------------------------------------------------
-- Test the newline parameter
local tmpl = [[
some list: $(T[1]:upper())
# for i = 2, #T do
,$(T[i]:upper())
# end
]]

local my_env = {
  ipairs = ipairs,
  T = {'one','two','three'}
}
local t, err = template.compile(tmpl, { debug = true, newline = "" })
local res, err, code = t:render(my_env)
--print(res, err, code)
asserteq(res, [[some list: ONE,TWO,THREE]])



--------------------------------------------------
-- Test template run-time error
local tmpl = [[
header: $("hello" * 10)
]]

local t, err = template.compile(tmpl, { debug = true, newline = "" })
local res, err, code = t:render()
--print(res, err, code)
assert(res == nil, "expected nil here because of the runtime error")
asserteq(type(err), "string")
asserteq(type(utils.load(code)), "function")



--------------------------------------------------
-- Test template run-time, doesn't fail on table value
-- table.concat fails if we insert a non-string (table) value
local tmpl = [[
header: $(myParam)
]]

local t, err = template.compile(tmpl, { debug = true, newline = "" })
local myParam = {}
local res, err, code = t:render( {myParam = myParam } ) -- insert a table
--print(res, err, code)
asserteq(res, "header: "..tostring(myParam))
asserteq(type(err), "nil")



--------------------------------------------------
-- Test template compile-time error
local tmpl = [[
header: $(this doesn't work)
]]

local my_env = {
  ipairs = ipairs,
  T = {'one','two','three'}
}
local t, err, code = template.compile(tmpl, { debug = true, newline = "" })
--print(t, err, code)
assert(t==nil, "expected t to be nil here because of the syntax error")
asserteq(type(err), "string")
asserteq(type(code), "string")



--------------------------------------------------
-- Test using template being a single static string
local tmpl = [[
<ul>
<p>a paragraph</p>
<p>a paragraph</p>
</ul>
]]

local t, err = template.compile(tmpl, { debug = true })
local res, err, code = t:render(my_env)
--print(res, err, code)

asserteq(res, [[<ul>
<p>a paragraph</p>
<p>a paragraph</p>
</ul>
]])
asserteq(code, [[return "<ul>\
<p>a paragraph</p>\
<p>a paragraph</p>\
</ul>\
"]])


print("template: success")
