local test = require 'pl.test'
local asserteq = test.asserteq

local compat = require "pl.compat"
local coroutine = require "coroutine"

local code_generator = coroutine.wrap(function()
    local result = {"ret", "urn \"Hello World!\""}
    for _,v in ipairs(result) do
        coroutine.yield(v)
    end
    coroutine.yield(nil)
end)

local f, err = compat.load(code_generator)
asserteq(err, nil)
asserteq(f(), "Hello World!")


-- package.searchpath
if compat.lua51 and not compat.jit then
  assert(package.searchpath("pl.compat", package.path):match("lua[/\\]pl[/\\]compat"))
  
  local path = "some/?/nice.path;another/?.path"
  local ok, err = package.searchpath("my.file.name", path, ".", "/")
  asserteq(err, "\tno file 'some/my/file/name/nice.path'\n\tno file 'another/my/file/name.path'")
  local ok, err = package.searchpath("my/file/name", path, "/", ".")
  asserteq(err, "\tno file 'some/my.file.name/nice.path'\n\tno file 'another/my.file.name.path'")
end