local test = require 'pl.test'
local LUA_VERSION = _VERSION
print(LUA_VERSION)

-- if STRICT is true, then M is distinct from _ENV, and ONLY contains
-- the exported functions!

local _ENV,M = require 'pl.import_into' (rawget(_G,'STRICT'))

function answer ()
    -- of course, you don't have the usual global environment available
    -- so define it as a local up above, or use utils.import(_G).

    local versioned_errors = {
        ["1"] = "attempt to call global 'print'",
        ["2"] = "attempt to call global 'print'",
        ["3"] = "attempt to call a nil value",
        ["4"] = "a nil value",
    }
    local expected = versioned_errors[LUA_VERSION:match("Lua 5.(%d)")]
    test.assertraise(function()
        print 'hello'
    end, expected)

    -- but all the Penlight modules are available
    return pretty.write(utils.split '10 20  30', '')
end

return M

