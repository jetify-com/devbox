local path = require 'pl.path'
local utils = require 'pl.utils'
local asserteq = require 'pl.test'.asserteq

local echo_lineending = "\n"
if path.is_windows then
    echo_lineending = " \n"
end

local function test_executeex(cmd, expected_successful, expected_retcode, expected_stdout, expected_stderr)
--print("\n"..cmd)
--print(os.execute(cmd))
--print(utils.executeex(cmd))
    local successful, retcode, stdout, stderr = utils.executeex(cmd)
    asserteq(successful, expected_successful)
    asserteq(retcode,    expected_retcode)
    asserteq(stdout,     expected_stdout)
    asserteq(stderr,     expected_stderr)
end

-- Check the return codes
if utils.is_windows then
    test_executeex("exit",        true,      0, "", "")
    test_executeex("exit 0",      true,      0, "", "")
    test_executeex("exit 1",      false,     1, "", "")
    test_executeex("exit 13",     false,    13, "", "")
    test_executeex("exit 255",    false,   255, "", "")
    test_executeex("exit 256",    false,   256, "", "")
    test_executeex("exit 257",    false,   257, "", "")
    test_executeex("exit 3809",   false,  3809, "", "")
    test_executeex("exit -1",     false,    -1, "", "")
    test_executeex("exit -13",    false,   -13, "", "")
    test_executeex("exit -255",   false,  -255, "", "")
    test_executeex("exit -256",   false,  -256, "", "")
    test_executeex("exit -257",   false,  -257, "", "")
    test_executeex("exit -3809",  false, -3809, "", "")
else
    test_executeex("exit",        true,      0, "", "")
    test_executeex("exit 0",      true,      0, "", "")
    test_executeex("exit 1",      false,     1, "", "")
    test_executeex("exit 13",     false,    13, "", "")
    test_executeex("exit 255",    false,   255, "", "")
    -- on posix anything other than 0-255 is undefined
    test_executeex("exit 256",    true,      0, "", "")
    test_executeex("exit 257",    false,     1, "", "")
    test_executeex("exit 3809",   false,   225, "", "")
end

-- Check output strings
test_executeex("echo stdout",                         true, 0, "stdout" .. echo_lineending, "")
test_executeex("(echo stderr 1>&2)",                  true, 0, "",                          "stderr" .. echo_lineending)
test_executeex("(echo stdout && (echo stderr 1>&2))", true, 0, "stdout" .. echo_lineending, "stderr" .. echo_lineending)

