--- Text processing utilities.
--
-- This provides a Template class (modeled after the same from the Python
-- libraries, see string.Template). It also provides similar functions to those
-- found in the textwrap module.
--
-- IMPORTANT: this module has been deprecated and will be removed in a future
-- version (2.0). The contents of this module have moved to the `pl.stringx`
-- module.
--
-- See  @{03-strings.md.String_Templates|the Guide}.
--
-- Dependencies: `pl.stringx`, `pl.utils`
-- @module pl.text

local utils = require("pl.utils")

utils.raise_deprecation {
  source = "Penlight " .. utils._VERSION,
  message = "the contents of module 'pl.text' has moved into 'pl.stringx'",
  version_removed = "2.0.0",
  deprecated_after = "1.11.0",
  no_trace = true,
}

return require "pl.stringx"
