local strict = require 'pl.strict'
local test = require 'pl.test'
local M = strict.module (...)

function M.answer ()
    Boo = false  -- fine, it's a declared global
    -- in strict mode, you cannot assign to globals if you aren't in main
    test.assertraise(function()
        Foo = true
    end," assign to undeclared global 'Foo'")
    return 42
end

function M.question ()
    return 'what is the answer to Life, the Universe and Everything?'
end

return M

