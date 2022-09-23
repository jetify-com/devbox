local app = require "pl.app"
local utils = require "pl.utils"
local path = require "pl.path"
local asserteq = require 'pl.test'.asserteq
local lfs = require("lfs")

local quote = utils.quote_arg

local _, cmd = app.lua()
cmd = cmd .. " " .. quote({"-e", "package.path=[[./lua/?.lua;./lua/?/init.lua;]]..package.path"})

local function run_script(s, fname)
    local tmpname = path.tmpname()
    if fname then
        tmpname = path.join(path.dirname(tmpname), fname)
    end
    assert(utils.writefile(tmpname, s))
    local success, code, stdout, stderr = utils.executeex(cmd.." "..tmpname)
    os.remove(tmpname)
    return success, code, stdout, stderr
end

do  -- app.script_name

    local success, code, stdout, stderr = run_script([[
        print(require("pl.app").script_name())
        ]],
        "justsomescriptname.lua")
    asserteq(stderr, "")
    asserteq(stdout:match("(justsome.+)$"), "justsomescriptname.lua\n")


    -- commandline, no scriptname
    local success, code, stdout, stderr = run_script([[
        arg[0] = nil  -- simulate no scriptname
        local name, err = require("pl.app").script_name()
        io.stdout:write(tostring(name))
        io.stderr:write(err)
        ]])
    assert(stderr:find("No script name found"))
    asserteq(stdout, "nil")


    -- commandline, no args table
    local success, code, stdout, stderr = run_script([[
        arg = nil  -- simulate no arg table
        local name, err = require("pl.app").script_name()
        io.stdout:write(tostring(name))
        io.stderr:write(err)
        ]])
    assert(stderr:find("No script name found"))
    asserteq(stdout, "nil")
end

do  -- app.require_here
    local cd = path.currentdir() --path.dirname(path.tmpname())

    -- plain script name
    local success, code, stdout, stderr = run_script([[
        arg[0] = "justsomescriptname.lua"
        local p = package.path
        require("pl.app").require_here()
        print(package.path:sub(1, -#p-1))
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    assert(stdout:find(path.normcase(cd.."/?.lua;"), 1, true))
    assert(stdout:find(path.normcase(cd.."/?/init.lua;"), 1, true))


    -- plain script name, with a relative base name
    local success, code, stdout, stderr = run_script([[
        arg[0] = "justsomescriptname.lua"
        local p = package.path
        require("pl.app").require_here("basepath/to/somewhere")
        print(package.path:sub(1, -#p-1))
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    assert(stdout:find(path.normcase(cd.."/basepath/to/somewhere/?.lua;"), 1, true))
    assert(stdout:find(path.normcase(cd.."/basepath/to/somewhere/?/init.lua;"), 1, true))


    -- plain script name, with an absolute base name
    local success, code, stdout, stderr = run_script([[
        arg[0] = "justsomescriptname.lua"
        local p = package.path
        require("pl.app").require_here("/basepath/to/somewhere")
        print(package.path:sub(1, -#p-1))
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    asserteq(stdout, path.normcase("/basepath/to/somewhere/?.lua;/basepath/to/somewhere/?/init.lua;\n"))


    -- scriptname with a relative path
    local success, code, stdout, stderr = run_script([[
        arg[0] = "relative/prefix/justsomescriptname.lua"
        local p = package.path
        require("pl.app").require_here()
        print(package.path:sub(1, -#p-1))
        os.exit()
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    assert(stdout:find(path.normcase(cd.."/relative/prefix/?.lua;"), 1, true))
    assert(stdout:find(path.normcase(cd.."/relative/prefix/?/init.lua;"), 1, true))


    -- script with an absolute path
    local success, code, stdout, stderr = run_script([[
        arg[0] = "/fixed/justsomescriptname.lua"
        local p = package.path
        require("pl.app").require_here()
        print(package.path:sub(1, -#p-1))
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    asserteq(stdout, path.normcase("/fixed/?.lua;/fixed/?/init.lua;\n"))

    -- symlinked script, check that we look beside the target of the link
    -- -- step 1: find ourselves
    local self = app.script_name()
    if not path.isabs(self) then self = path.join(cd,self) end
    local tadir = path.normcase(path.join(path.dirname(self),"test-app"))
    -- -- step 2: create a link to our helper script
    local scrl = path.tmpname()
    local linkdir = path.normcase(path.dirname(scrl))
    os.remove(scrl)
    assert(lfs.link(path.join(tadir,"require_here-link-target.lua"), scrl, true))
    -- -- step 3: check that we look next to ourselves
    local success, code, stdout, stderr = utils.executeex(cmd.." "..scrl)
    stdout = path.normcase(stdout)
    assert(stdout:find(path.normcase(path.join(tadir, "?.lua;")), 1, true))
    assert(stdout:find(path.normcase(path.join(tadir, "?/init.lua;")), 1, true))
    assert(not stdout:find(path.normcase(path.join(linkdir, "?.lua;")), 1, true))
    assert(not stdout:find(path.normcase(path.join(linkdir, "?/init.lua;")), 1, true))
    -- -- step 4: ... but not if we turn on nofollow
    local success, code, stdout, stderr = utils.executeex(cmd.." "..scrl.." x")
    stdout = path.normcase(stdout)
    assert(not stdout:find(path.normcase(path.join(tadir, "?.lua;")), 1, true))
    assert(not stdout:find(path.normcase(path.join(tadir, "?/init.lua;")), 1, true))
    assert(stdout:find(path.normcase(path.join(linkdir, "?.lua;")), 1, true))
    assert(stdout:find(path.normcase(path.join(linkdir, "?/init.lua;")), 1, true))
    os.remove(scrl)

end


do  -- app.appfile
    local success, code, stdout, stderr = run_script([[
        arg[0] = "some/path/justsomescriptname_for_penlight_testing.lua"
        print(require("pl.app").appfile("filename.data"))
        ]])
    asserteq(stderr, "")
    stdout = path.normcase(stdout)
    local fname = path.normcase(path.expanduser("~/.justsomescriptname_for_penlight_testing/filename.data"))
    asserteq(stdout, fname .."\n")
    assert(path.isdir(path.dirname(fname)))
    path.rmdir(path.dirname(fname))

end


do  -- app.lua
    local success, code, stdout, stderr = run_script([[
        arg[0] = "justsomescriptname.lua"
        local a,b = require("pl.app").lua()
        print(a)
        ]])
    asserteq(stderr, "")
    asserteq(stdout, cmd .."\n")

end


do -- app.parse_args

    -- no value specified
    local args = utils.split("-a -b")
    local t,s = app.parse_args(args, { a = true})
    asserteq(t, nil)
    asserteq(s, "no value for 'a'")


    -- flag that take a value, space separated
    local args = utils.split("-a -b value -c")
    local t,s = app.parse_args(args, { b = true})
    asserteq(t, {
        a = true,
        b = "value",
        c = true,
    })
    asserteq(s, {})


    -- flag_with_values specified as a list
    local args = utils.split("-a -b value -c")
    local t,s = app.parse_args(args, { "b" })
    asserteq(t, {
        a = true,
        b = "value",
        c = true,
    })
    asserteq(s, {})


    -- flag_with_values missing value at end
    local args = utils.split("-a -b")
    local t,s = app.parse_args(args, { "b" })
    asserteq(t, nil)
    asserteq(s, "no value for 'b'")


    -- error on an unknown flag
    local args = utils.split("-a -b value -c")
    local t,s = app.parse_args(args, { b = true }, { "b", "c" })
    asserteq(t, nil)
    asserteq(s, "unknown flag 'a'")


    -- flag that doesn't take a value
    local args = utils.split("-a -b:value")
    local t,s = app.parse_args(args, {})
    asserteq(t, {
       ["a"] = true,
       ["b"] = "value"
    })
    asserteq(s, {})


    -- correctly parsed values, spaces, :, =, and multiple : or =
    local args = utils.split("-a value -b value:one=two -c=value2:2")
    local t,s = app.parse_args(args, { "a", "b", "c" })
    asserteq(t, {
       ["a"] = "value",
       ["b"] = "value:one=two",
       ["c"] = "value2:2",
    })
    asserteq(s, {})


    -- many values, duplicates, and parameters mixed
    local args = utils.split(
      "-a -b -cde --long1 --ff:ffvalue --gg=ggvalue -h:hvalue -i=ivalue " ..
      "-i=2ndvalue param -i:3rdvalue -j1 -k2 -1:hello remaining values")
    local t,s = app.parse_args(args)
    asserteq({
        i = "3rdvalue",
        ["1"] = "hello",
        ff = "ffvalue",
        long1 = true,
        c = true,
        b = true,
        gg = "ggvalue",
        j = "1",
        k = "2",
        d = true,
        h = "hvalue",
        a = true,
        e = true
      }, t)
    asserteq({
        "param",
        "remaining",
        "values"
      }, s)


    -- specify valid flags and aliasses
    local args = utils.split("-a -b value -e -f3")
    local t,s = app.parse_args(args,
      {
        "b",
        f = true,
      }, {
        bully = "b",   -- b with value will be reported as 'bully', alias as string
        a = true,      -- hash-type value
        c = { "d", "e" }, -- e will be reported as c, aliasses as list/table
      })
    asserteq(t, {
        a = true,
        bully = "value",
        c = true,
        f = "3",
    })
    asserteq(s, {})


    -- error on an unknown flag, in a chain of short ones
    local args = utils.split("-b value -cd")
    local t,s = app.parse_args(args, { b = true }, { "b", "c" })
    asserteq(t, nil)
    asserteq(s, "unknown flag 'd'")


    -- flag, in a chain of short ones, gets converted to alias
    local args = utils.split("-dbc")
    local t,s = app.parse_args(args, nil, { "d", full_name = "b", "c" })
    asserteq(t, {
        full_name = true,  -- specified as b in a chain of short ones
        c = true,
        d = true,
    })
    asserteq(s, {})

end
