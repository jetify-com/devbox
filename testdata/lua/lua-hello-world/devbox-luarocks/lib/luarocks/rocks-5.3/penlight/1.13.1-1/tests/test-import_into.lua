local test = require 'pl.test'
local utils = require 'pl.utils'

require 'pl.app'.require_here 'lua'

if not utils.lua51 then
    --- look at lua/mod52.lua
    local m = require 'mod52'
    test.asserteq(m.answer(),'{"10","20","30"}')
    assert(m.utils)  -- !! implementation is leaky!

    -- that's a bugger. However, if 'pl.import_into' is passed true,
    -- then the returned module will _only_ contain the newly defined functions
    -- So reload after setting the global STRICT
    package.loaded.mod52 = nil
    STRICT = true
    m = require 'mod52'
    assert (m.answer) -- as before
    assert (not m.utils) -- cool! No underwear showing
end

local pl = require 'pl.import_into' ()

assert(pl.utils)
assert(pl.tablex)
assert(pl.data)
assert(not _G.utils)
assert(not _G.tablex)
assert(not _G.data)

require 'pl.import_into'(_G)
assert(_G.utils)
assert(_G.tablex)
assert(_G.data)

require 'pl.import_into'(_G)
assert(_G.utils)
assert(_G.tablex)
assert(_G.data)
