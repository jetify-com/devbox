local asserteq = require('pl.test').asserteq
local lexer = require 'pl.lexer'
local seq = require 'pl.seq'
local List = require('pl.List')
local open = require('pl.stringio').open
local copy2 = seq.copy2

local function test_scan(str, filter, options, expected_tokens, lang)
    local matches
    if lang then
        matches, filter = filter, options
    else
        lang = 'scan'
    end

    asserteq(copy2(lexer[lang](str, matches, filter, options)), expected_tokens)
    if lang == 'scan' then
        asserteq(copy2(lexer[lang](open(str), matches, filter, options)), expected_tokens)
    end
end

local s = '20 = hello'
test_scan(s, {space=false}, {number=false}, {
    {'number', '20'}, {'space', ' '}, {'=', '='}, {'space', ' '}, {'iden', 'hello'}
})
test_scan(s, {space=true}, {number=true}, {
    {'number', 20}, {'=', '='}, {'iden', 'hello'}
})
s = [[    'help'  "help" "dolly you're fine" "a \"quote\" here"]]
test_scan(s, nil, nil, {
    {'string', 'help'}, {'string', 'help'},
    {'string', "dolly you're fine"}, {'string', 'a \\\"quote\\\" here'}  -- Escapes are preserved literally.
})
test_scan([[\abc\]], nil, nil, {
    {'\\', '\\'}, {'iden', 'abc'}, {'\\', '\\'}
})
test_scan([["" ""]], nil, nil, {
    {'string', ''}, {'string', ''}
})
test_scan([["abc" "def\\"]], nil, nil, {
    {'string', 'abc'}, {'string', 'def\\\\'}
})
test_scan([["abc\\" "def"]], nil, nil, {
    {'string', 'abc\\\\'}, {'string', 'def'}
})
test_scan([["abc\\\" "]], nil, nil, {
    {'string', 'abc\\\\\\" '}
})

local function test_roundtrip(str)
    test_scan(str, {}, {string=false}, {{'string', str}})
end

test_roundtrip [["hello\\"]]
test_roundtrip [["hello\"dolly"]]
test_roundtrip [['hello\'dolly']]
test_roundtrip [['']]
test_roundtrip [[""]]

test_scan('test(20 and a > b)', nil, nil, {
    {'iden', 'test'}, {'(', '('}, {'number', 20}, {'keyword', 'and'},
    {'iden', 'a'}, {'>', '>'}, {'iden', 'b'}, {')', ')'}
}, 'lua')
test_scan('10+2.3', nil, nil, {
    {'number', 10}, {'+', '+'}, {'number', 2.3}
}, 'lua')

local txt = [==[
-- comment
--[[
block
comment
]][[
hello dammit
]][[hello]]
]==]

test_scan(txt, {}, nil, {
    {'comment', '-- comment\n'},
    {'comment', '--[[\nblock\ncomment\n]]'},
    {'string', 'hello dammit\n'},
    {'string', 'hello'},
    {'space', '\n'}
}, 'lua')

local lines = [[
for k,v in pairs(t) do
    if type(k) == 'number' then
        print(v) -- array-like case
    else
        print(k,v)
    end -- if
end
]]

local ls = List()
for tp,val in lexer.lua(lines,{space=true,comments=true}) do
    assert(tp ~= 'space' and tp ~= 'comment')
    if tp == 'keyword' then ls:append(val) end
end
asserteq(ls,List{'for','in','do','if','then','else','end','end'})

txt = [[
// comment
/* a long
set of words */ // more
]]

test_scan(txt, {}, nil, {
    {'comment', '// comment\n'},
    {'comment', '/* a long\nset of words */'},
    {'space', ' '},
    {'comment', '// more\n'}
}, 'cpp')

test_scan([['' "" " \\" '\'' "'"]], nil, nil, {
    {'char', ''}, -- Char literals with no or more than one characters are not a lexing error.
    {'string', ''},
    {'string', ' \\\\'},
    {'char', "\\'"},
    {'string', "'"}
}, 'cpp')

local iter = lexer.lua([[
foo
bar
]])

asserteq(lexer.lineno(iter), 0)
iter()
asserteq(lexer.lineno(iter), 1)
asserteq(lexer.lineno(iter), 1)
iter()
asserteq(lexer.lineno(iter), 2)
iter()
asserteq(lexer.lineno(iter), 3)
iter()
iter()
asserteq(lexer.lineno(iter), 3)

do  -- numbers without leading zero; ".123"
  local s = 'hello = +.234'
  test_scan(s, {space=true}, {number=true}, {
    {'iden', 'hello'}, {'=', '='}, {'number', .234}
  })
end
