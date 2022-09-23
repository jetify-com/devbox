local T = require 'pl.text'
local utils = require 'pl.utils'
local Template = T.Template
local asserteq = require 'pl.test'.asserteq
local OrderedMap = require 'pl.OrderedMap'
local template = require 'pl.template'

local t = [[
# for i = 1,3 do
    print($(i+1))
# end
]]

asserteq(template.substitute(t),[[
    print(2)
    print(3)
    print(4)
]])

t = [[
> for i = 1,3 do
    print(${i+1})
> end
]]

asserteq(template.substitute(t,{_brackets='{}',_escape='>'}),[[
    print(2)
    print(3)
    print(4)
]])

t = [[
#@ for i = 1,3 do
    print(@{i+1})
#@ end
]]

asserteq(template.substitute(t,{_brackets='{}',_escape='#@',_inline_escape='@'}),[[
    print(2)
    print(3)
    print(4)
]])

--- iteration using pairs is usually unordered. But using OrderedMap
--- we can get the exact original ordering.

t = [[
# for k,v in pairs(T) do
    "$(k)", -- $(v)
# end
]]

if utils.lua51 then
    -- easy enough to define a general pairs in Lua 5.1
    local rawpairs = pairs
    function pairs(t)
        local mt = getmetatable(t)
        local f = mt and mt.__pairs
        if f then
            return f(t)
        else
            return rawpairs(t)
        end
    end
end


local Tee = OrderedMap{{Dog = 'Bonzo'}, {Cat = 'Felix'}, {Lion = 'Leo'}}

-- note that the template will also look up global functions using _parent
asserteq(template.substitute(t,{T=Tee,_parent=_G}),[[
    "Dog", -- Bonzo
    "Cat", -- Felix
    "Lion", -- Leo
]])

