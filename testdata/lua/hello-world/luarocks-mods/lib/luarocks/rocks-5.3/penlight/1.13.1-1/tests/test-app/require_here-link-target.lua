-- This file is used as a symbolic link target in test-app.lua, to verify the
-- behaviors of pl.app.require_here()

local p = package.path
require("pl.app").require_here(nil, #arg > 0)
print(package.path:sub(1, -#p-1))
