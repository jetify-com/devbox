--- Python-style URL quoting library.
--
-- @module pl.url

local url = {}

local function quote_char(c)
    return string.format("%%%02X", string.byte(c))
end

--- Quote the url, replacing special characters using the '%xx' escape.
-- @string s the string
-- @bool quote_plus Also escape slashes and replace spaces by plus signs.
-- @return The quoted string, or if `s` wasn't a string, just plain unaltered `s`.
function url.quote(s, quote_plus)
    if type(s) ~= "string" then
        return s
    end

    s = s:gsub("\n", "\r\n")
    s = s:gsub("([^A-Za-z0-9 %-_%./])", quote_char)
    if quote_plus then
        s = s:gsub(" ", "+")
        s = s:gsub("/", quote_char)
    else
        s = s:gsub(" ", "%%20")
    end

    return s
end

local function unquote_char(h)
    return string.char(tonumber(h, 16))
end

--- Unquote the url, replacing '%xx' escapes and plus signs.
-- @string s the string
-- @return The unquoted string, or if `s` wasn't a string, just plain unaltered `s`.
function url.unquote(s)
    if type(s) ~= "string" then
        return s
    end

    s = s:gsub("+", " ")
    s = s:gsub("%%(%x%x)", unquote_char)
    s = s:gsub("\r\n", "\n")

    return s
end

return url
