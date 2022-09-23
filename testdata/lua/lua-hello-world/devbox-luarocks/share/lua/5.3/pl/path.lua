--- Path manipulation and file queries.
--
-- This is modelled after Python's os.path library (10.1); see @{04-paths.md|the Guide}.
--
-- NOTE: the functions assume the paths being dealt with to originate
-- from the OS the application is running on. Windows drive letters are not
-- to be used when running on a Unix system for example. The one exception
-- is Windows paths to allow both forward and backward slashes (since Lua
-- also accepts those)
--
-- Dependencies: `pl.utils`, `lfs`
-- @module pl.path

-- imports and locals
local _G = _G
local sub = string.sub
local getenv = os.getenv
local tmpnam = os.tmpname
local package = package
local append, concat, remove = table.insert, table.concat, table.remove
local utils = require 'pl.utils'
local assert_string,raise = utils.assert_string,utils.raise

local res,lfs = _G.pcall(_G.require,'lfs')
if not res then
    error("pl.path requires LuaFileSystem")
end

local attrib = lfs.attributes
local currentdir = lfs.currentdir
local link_attrib = lfs.symlinkattributes

local path = {}

local function err_func(name, param, err, code)
  local ret = ("%s failed"):format(tostring(name))
  if param ~= nil then
    ret = ret .. (" for '%s'"):format(tostring(param))
  end
  ret = ret .. (": %s"):format(tostring(err))
  if code ~= nil then
    ret = ret .. (" (code %s)"):format(tostring(code))
  end
  return ret
end

--- Lua iterator over the entries of a given directory.
-- Implicit link to [`luafilesystem.dir`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function dir
path.dir = lfs.dir

--- Creates a directory.
-- Implicit link to [`luafilesystem.mkdir`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function mkdir
path.mkdir = function(d)
  local ok, err, code = lfs.mkdir(d)
  if not ok then
    return ok, err_func("mkdir", d, err, code), code
  end
  return ok, err, code
end

--- Removes a directory.
-- Implicit link to [`luafilesystem.rmdir`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function rmdir
path.rmdir = function(d)
  local ok, err, code = lfs.rmdir(d)
  if not ok then
    return ok, err_func("rmdir", d, err, code), code
  end
  return ok, err, code
end

--- Gets attributes.
-- Implicit link to [`luafilesystem.attributes`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function attrib
path.attrib = function(d, r)
  local ok, err, code = attrib(d, r)
  if not ok then
    return ok, err_func("attrib", d, err, code), code
  end
  return ok, err, code
end

--- Get the working directory.
-- Implicit link to [`luafilesystem.currentdir`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function currentdir
path.currentdir = function()
  local ok, err, code = currentdir()
  if not ok then
    return ok, err_func("currentdir", nil, err, code), code
  end
  return ok, err, code
end

--- Gets symlink attributes.
-- Implicit link to [`luafilesystem.symlinkattributes`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function link_attrib
path.link_attrib = function(d, r)
  local ok, err, code = link_attrib(d, r)
  if not ok then
    return ok, err_func("link_attrib", d, err, code), code
  end
  return ok, err, code
end

--- Changes the working directory.
-- On Windows, if a drive is specified, it also changes the current drive. If
-- only specifying the drive, it will only switch drive, but not modify the path.
-- Implicit link to [`luafilesystem.chdir`](https://keplerproject.github.io/luafilesystem/manual.html#reference)
-- @function chdir
path.chdir = function(d)
  local ok, err, code = lfs.chdir(d)
  if not ok then
    return ok, err_func("chdir", d, err, code), code
  end
  return ok, err, code
end

--- is this a directory?
-- @string P A file path
function path.isdir(P)
    assert_string(1,P)
    return attrib(P,'mode') == 'directory'
end

--- is this a file?
-- @string P A file path
function path.isfile(P)
    assert_string(1,P)
    return attrib(P,'mode') == 'file'
end

-- is this a symbolic link?
-- @string P A file path
function path.islink(P)
    assert_string(1,P)
    if link_attrib then
        return link_attrib(P,'mode')=='link'
    else
        return false
    end
end

--- return size of a file.
-- @string P A file path
function path.getsize(P)
    assert_string(1,P)
    return attrib(P,'size')
end

--- does a path exist?
-- @string P A file path
-- @return the file path if it exists (either as file, directory, socket, etc), nil otherwise
function path.exists(P)
    assert_string(1,P)
    return attrib(P,'mode') ~= nil and P
end

--- Return the time of last access as the number of seconds since the epoch.
-- @string P A file path
function path.getatime(P)
    assert_string(1,P)
    return attrib(P,'access')
end

--- Return the time of last modification as the number of seconds since the epoch.
-- @string P A file path
function path.getmtime(P)
    assert_string(1,P)
    return attrib(P,'modification')
end

---Return the system's ctime as the number of seconds since the epoch.
-- @string P A file path
function path.getctime(P)
    assert_string(1,P)
    return path.attrib(P,'change')
end


local function at(s,i)
    return sub(s,i,i)
end

path.is_windows = utils.is_windows

local sep, other_sep, seps
-- constant sep is the directory separator for this platform.
-- constant dirsep is the separator in the PATH environment variable
if path.is_windows then
    path.sep = '\\'; other_sep = '/'
    path.dirsep = ';'
    seps = { ['/'] = true, ['\\'] = true }
else
    path.sep = '/'
    path.dirsep = ':'
    seps = { ['/'] = true }
end
sep = path.sep

--- are we running Windows?
-- @class field
-- @name path.is_windows

--- path separator for this platform.
-- @class field
-- @name path.sep

--- separator for PATH for this platform
-- @class field
-- @name path.dirsep

--- given a path, return the directory part and a file part.
-- if there's no directory part, the first value will be empty
-- @string P A file path
-- @return directory part
-- @return file part
-- @usage
-- local dir, file = path.splitpath("some/dir/myfile.txt")
-- assert(dir == "some/dir")
-- assert(file == "myfile.txt")
--
-- local dir, file = path.splitpath("some/dir/")
-- assert(dir == "some/dir")
-- assert(file == "")
--
-- local dir, file = path.splitpath("some_dir")
-- assert(dir == "")
-- assert(file == "some_dir")
function path.splitpath(P)
    assert_string(1,P)
    local i = #P
    local ch = at(P,i)
    while i > 0 and ch ~= sep and ch ~= other_sep do
        i = i - 1
        ch = at(P,i)
    end
    if i == 0 then
        return '',P
    else
        return sub(P,1,i-1), sub(P,i+1)
    end
end

--- return an absolute path.
-- @string P A file path
-- @string[opt] pwd optional start path to use (default is current dir)
function path.abspath(P,pwd)
    assert_string(1,P)
    if pwd then assert_string(2,pwd) end
    local use_pwd = pwd ~= nil
    if not use_pwd and not currentdir() then return P end
    P = P:gsub('[\\/]$','')
    pwd = pwd or currentdir()
    if not path.isabs(P) then
        P = path.join(pwd,P)
    elseif path.is_windows and not use_pwd and at(P,2) ~= ':' and at(P,2) ~= '\\' then
        P = pwd:sub(1,2)..P -- attach current drive to path like '\\fred.txt'
    end
    return path.normpath(P)
end

--- given a path, return the root part and the extension part.
-- if there's no extension part, the second value will be empty
-- @string P A file path
-- @treturn string root part (everything upto the "."", maybe empty)
-- @treturn string extension part (including the ".", maybe empty)
-- @usage
-- local file_path, ext = path.splitext("/bonzo/dog_stuff/cat.txt")
-- assert(file_path == "/bonzo/dog_stuff/cat")
-- assert(ext == ".txt")
--
-- local file_path, ext = path.splitext("")
-- assert(file_path == "")
-- assert(ext == "")
function path.splitext(P)
    assert_string(1,P)
    local i = #P
    local ch = at(P,i)
    while i > 0 and ch ~= '.' do
        if seps[ch] then
            return P,''
        end
        i = i - 1
        ch = at(P,i)
    end
    if i == 0 then
        return P,''
    else
        return sub(P,1,i-1),sub(P,i)
    end
end

--- return the directory part of a path
-- @string P A file path
-- @treturn string everything before the last dir-separator
-- @see splitpath
-- @usage
-- path.dirname("/some/path/file.txt")   -- "/some/path"
-- path.dirname("file.txt")              -- "" (empty string)
function path.dirname(P)
    assert_string(1,P)
    local p1 = path.splitpath(P)
    return p1
end

--- return the file part of a path
-- @string P A file path
-- @treturn string
-- @see splitpath
-- @usage
-- path.basename("/some/path/file.txt")  -- "file.txt"
-- path.basename("/some/path/file/")     -- "" (empty string)
function path.basename(P)
    assert_string(1,P)
    local _,p2 = path.splitpath(P)
    return p2
end

--- get the extension part of a path.
-- @string P A file path
-- @treturn string
-- @see splitext
-- @usage
-- path.extension("/some/path/file.txt") -- ".txt"
-- path.extension("/some/path/file_txt") -- "" (empty string)
function path.extension(P)
    assert_string(1,P)
    local _,p2 = path.splitext(P)
    return p2
end

--- is this an absolute path?
-- @string P A file path
-- @usage
-- path.isabs("hello/path")    -- false
-- path.isabs("/hello/path")   -- true
-- -- Windows;
-- path.isabs("hello\path")    -- false
-- path.isabs("\hello\path")   -- true
-- path.isabs("C:\hello\path") -- true
-- path.isabs("C:hello\path")  -- false
function path.isabs(P)
    assert_string(1,P)
    if path.is_windows and at(P,2) == ":" then
        return seps[at(P,3)] ~= nil
    end
    return seps[at(P,1)] ~= nil
end

--- return the path resulting from combining the individual paths.
-- if the second (or later) path is absolute, we return the last absolute path (joined with any non-absolute paths following).
-- empty elements (except the last) will be ignored.
-- @string p1 A file path
-- @string p2 A file path
-- @string ... more file paths
-- @treturn string the combined path
-- @usage
-- path.join("/first","second","third")   -- "/first/second/third"
-- path.join("first","second/third")      -- "first/second/third"
-- path.join("/first","/second","third")  -- "/second/third"
function path.join(p1,p2,...)
    assert_string(1,p1)
    assert_string(2,p2)
    if select('#',...) > 0 then
        local p = path.join(p1,p2)
        local args = {...}
        for i = 1,#args do
            assert_string(i,args[i])
            p = path.join(p,args[i])
        end
        return p
    end
    if path.isabs(p2) then return p2 end
    local endc = at(p1,#p1)
    if endc ~= path.sep and endc ~= other_sep and endc ~= "" then
        p1 = p1..path.sep
    end
    return p1..p2
end

--- normalize the case of a pathname. On Unix, this returns the path unchanged,
-- for Windows it converts;
--
-- * the path to lowercase
-- * forward slashes to backward slashes
-- @string P A file path
-- @usage path.normcase("/Some/Path/File.txt")
-- -- Windows: "\some\path\file.txt"
-- -- Others : "/Some/Path/File.txt"
function path.normcase(P)
    assert_string(1,P)
    if path.is_windows then
        return P:gsub('/','\\'):lower()
    else
        return P
    end
end

--- normalize a path name.
-- `A//B`, `A/./B`, and `A/foo/../B` all become `A/B`.
--
-- An empty path results in '.'.
-- @string P a file path
function path.normpath(P)
    assert_string(1,P)
    -- Split path into anchor and relative path.
    local anchor = ''
    if path.is_windows then
        if P:match '^\\\\' then -- UNC
            anchor = '\\\\'
            P = P:sub(3)
        elseif seps[at(P, 1)] then
            anchor = '\\'
            P = P:sub(2)
        elseif at(P, 2) == ':' then
            anchor = P:sub(1, 2)
            P = P:sub(3)
            if seps[at(P, 1)] then
                anchor = anchor..'\\'
                P = P:sub(2)
            end
        end
        P = P:gsub('/','\\')
    else
        -- According to POSIX, in path start '//' and '/' are distinct,
        -- but '///+' is equivalent to '/'.
        if P:match '^//' and at(P, 3) ~= '/' then
            anchor = '//'
            P = P:sub(3)
        elseif at(P, 1) == '/' then
            anchor = '/'
            P = P:match '^/*(.*)$'
        end
    end
    local parts = {}
    for part in P:gmatch('[^'..sep..']+') do
        if part == '..' then
            if #parts ~= 0 and parts[#parts] ~= '..' then
                remove(parts)
            else
                append(parts, part)
            end
        elseif part ~= '.' then
            append(parts, part)
        end
    end
    P = anchor..concat(parts, sep)
    if P == '' then P = '.' end
    return P
end

--- relative path from current directory or optional start point
-- @string P a path
-- @string[opt] start optional start point (default current directory)
function path.relpath (P,start)
    assert_string(1,P)
    if start then assert_string(2,start) end
    local split,min,append = utils.split, math.min, table.insert
    P = path.abspath(P,start)
    start = start or currentdir()
    local compare
    if path.is_windows then
        P = P:gsub("/","\\")
        start = start:gsub("/","\\")
        compare = function(v) return v:lower() end
    else
        compare = function(v) return v end
    end
    local startl, Pl = split(start,sep), split(P,sep)
    local n = min(#startl,#Pl)
    if path.is_windows and n > 0 and at(Pl[1],2) == ':' and Pl[1] ~= startl[1] then
        return P
    end
    local k = n+1 -- default value if this loop doesn't bail out!
    for i = 1,n do
        if compare(startl[i]) ~= compare(Pl[i]) then
            k = i
            break
        end
    end
    local rell = {}
    for i = 1, #startl-k+1 do rell[i] = '..' end
    if k <= #Pl then
        for i = k,#Pl do append(rell,Pl[i]) end
    end
    return table.concat(rell,sep)
end


--- Replace a starting '~' with the user's home directory.
-- In windows, if HOME isn't set, then USERPROFILE is used in preference to
-- HOMEDRIVE HOMEPATH. This is guaranteed to be writeable on all versions of Windows.
-- @string P A file path
function path.expanduser(P)
    assert_string(1,P)
    if at(P,1) == '~' then
        local home = getenv('HOME')
        if not home then -- has to be Windows
            home = getenv 'USERPROFILE' or (getenv 'HOMEDRIVE' .. getenv 'HOMEPATH')
        end
        return home..sub(P,2)
    else
        return P
    end
end


---Return a suitable full path to a new temporary file name.
-- unlike os.tmpname(), it always gives you a writeable path (uses TEMP environment variable on Windows)
function path.tmpname ()
    local res = tmpnam()
    -- On Windows if Lua is compiled using MSVC14 os.tmpname
    -- already returns an absolute path within TEMP env variable directory,
    -- no need to prepend it.
    if path.is_windows and not res:find(':') then
        res = getenv('TEMP')..res
    end
    return res
end

--- return the largest common prefix path of two paths.
-- @string path1 a file path
-- @string path2 a file path
-- @return the common prefix (Windows: separators will be normalized, casing will be original)
function path.common_prefix (path1,path2)
    assert_string(1,path1)
    assert_string(2,path2)
    -- get them in order!
    if #path1 > #path2 then path2,path1 = path1,path2 end
    local compare
    if path.is_windows then
        path1 = path1:gsub("/", "\\")
        path2 = path2:gsub("/", "\\")
        compare = function(v) return v:lower() end
    else
        compare = function(v) return v end
    end
    for i = 1,#path1 do
        if compare(at(path1,i)) ~= compare(at(path2,i)) then
            local cp = path1:sub(1,i-1)
            if at(path1,i-1) ~= sep then
                cp = path.dirname(cp)
            end
            return cp
        end
    end
    if at(path2,#path1+1) ~= sep then path1 = path.dirname(path1) end
    return path1
    --return ''
end

--- return the full path where a particular Lua module would be found.
-- Both package.path and package.cpath is searched, so the result may
-- either be a Lua file or a shared library.
-- @string mod name of the module
-- @return on success: path of module, lua or binary
-- @return on error: nil, error string listing paths tried
function path.package_path(mod)
    assert_string(1,mod)
    local res, err1, err2
    res, err1 = package.searchpath(mod,package.path)
    if res then return res,true end
    res, err2 = package.searchpath(mod,package.cpath)
    if res then return res,false end
    return raise ('cannot find module on path\n' .. err1 .. "\n" .. err2)
end


---- finis -----
return path
