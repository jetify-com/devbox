
-- require "luarocks.loader"

require 'pl'

local args = lapp [[
  -n, --name (string) name 
]]

print("hello "..args.name)
