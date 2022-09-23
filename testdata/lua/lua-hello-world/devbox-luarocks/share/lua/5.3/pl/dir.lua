--- Listing files in directories and creating/removing directory paths.
--
-- Dependencies: `pl.utils`, `pl.path`
--
-- Soft Dependencies: `alien`, `ffi` (either are used on Windows for copying/moving files)
-- @module pl.dir

local utils = require 'pl.utils'
local path = require 'pl.path'
local is_windows = path.is_windows
local ldir = path.dir
local mkdir = path.mkdir
local rmdir = path.rmdir
local sub = string.sub
local os,pcall,ipairs,pairs,require,setmetatable = os,pcall,ipairs,pairs,require,setmetatable
local remove = os.remove
local append = table.insert
local assert_arg,assert_string,raise = utils.assert_arg,utils.assert_string,utils.raise

local exists, isdir = path.exists, path.isdir
local sep = path.sep

local dir = {}

local function makelist(l)
    return setmetatable(l, require('pl.List'))
end

local function assert_dir (n,val)
    assert_arg(n,val,'string',path.isdir,'not a directory',4)
end

local function filemask(mask)
    mask = utils.escape(path.normcase(mask))
    return '^'..mask:gsub('%%%*','.*'):gsub('%%%?','.')..'$'
end

--- Test whether a file name matches a shell pattern.
-- Both parameters are case-normalized if operating system is
-- case-insensitive.
-- @string filename A file name.
-- @string pattern A shell pattern. The only special characters are
-- `'*'` and `'?'`: `'*'` matches any sequence of characters and
-- `'?'` matches any single character.
-- @treturn bool
-- @raise dir and mask must be strings
function dir.fnmatch(filename,pattern)
    assert_string(1,filename)
    assert_string(2,pattern)
    return path.normcase(filename):find(filemask(pattern)) ~= nil
end

--- Return a list of all file names within an array which match a pattern.
-- @tab filenames An array containing file names.
-- @string pattern A shell pattern (see `fnmatch`).
-- @treturn List(string) List of matching file names.
-- @raise dir and mask must be strings
function dir.filter(filenames,pattern)
    assert_arg(1,filenames,'table')
    assert_string(2,pattern)
    local res = {}
    local mask = filemask(pattern)
    for i,f in ipairs(filenames) do
        if path.normcase(f):find(mask) then append(res,f) end
    end
    return makelist(res)
end

local function _listfiles(dirname,filemode,match)
    local res = {}
    local check = utils.choose(filemode,path.isfile,path.isdir)
    if not dirname then dirname = '.' end
    for f in ldir(dirname) do
        if f ~= '.' and f ~= '..' then
            local p = path.join(dirname,f)
            if check(p) and (not match or match(f)) then
                append(res,p)
            end
        end
    end
    return makelist(res)
end

--- return a list of all files in a directory which match a shell pattern.
-- @string[opt='.'] dirname A directory.
-- @string[opt] mask A shell pattern (see `fnmatch`). If not given, all files are returned.
-- @treturn {string} list of files
-- @raise dirname and mask must be strings
function dir.getfiles(dirname,mask)
    dirname = dirname or '.'
    assert_dir(1,dirname)
    if mask then assert_string(2,mask) end
    local match
    if mask then
        mask = filemask(mask)
        match = function(f)
            return path.normcase(f):find(mask)
        end
    end
    return _listfiles(dirname,true,match)
end

--- return a list of all subdirectories of the directory.
-- @string[opt='.'] dirname A directory.
-- @treturn {string} a list of directories
-- @raise dir must be a valid directory
function dir.getdirectories(dirname)
    dirname = dirname or '.'
    assert_dir(1,dirname)
    return _listfiles(dirname,false)
end

local alien,ffi,ffi_checked,CopyFile,MoveFile,GetLastError,win32_errors,cmd_tmpfile

local function execute_command(cmd,parms)
   if not cmd_tmpfile then cmd_tmpfile = path.tmpname () end
   local err = path.is_windows and ' > ' or ' 2> '
    cmd = cmd..' '..parms..err..utils.quote_arg(cmd_tmpfile)
    local ret = utils.execute(cmd)
    if not ret then
        local err = (utils.readfile(cmd_tmpfile):gsub('\n(.*)',''))
        remove(cmd_tmpfile)
        return false,err
    else
        remove(cmd_tmpfile)
        return true
    end
end

local function find_ffi_copyfile ()
    if not ffi_checked then
        ffi_checked = true
        local res
        res,alien = pcall(require,'alien')
        if not res then
            alien = nil
            res, ffi = pcall(require,'ffi')
        end
        if not res then
            ffi = nil
            return
        end
    else
        return
    end
    if alien then
        -- register the Win32 CopyFile and MoveFile functions
        local kernel = alien.load('kernel32.dll')
        CopyFile = kernel.CopyFileA
        CopyFile:types{'string','string','int',ret='int',abi='stdcall'}
        MoveFile = kernel.MoveFileA
        MoveFile:types{'string','string',ret='int',abi='stdcall'}
        GetLastError = kernel.GetLastError
        GetLastError:types{ret ='int', abi='stdcall'}
    elseif ffi then
        ffi.cdef [[
            int CopyFileA(const char *src, const char *dest, int iovr);
            int MoveFileA(const char *src, const char *dest);
            int GetLastError();
        ]]
        CopyFile = ffi.C.CopyFileA
        MoveFile = ffi.C.MoveFileA
        GetLastError = ffi.C.GetLastError
    end
    win32_errors = {
        ERROR_FILE_NOT_FOUND    =         2,
        ERROR_PATH_NOT_FOUND    =         3,
        ERROR_ACCESS_DENIED    =          5,
        ERROR_WRITE_PROTECT    =          19,
        ERROR_BAD_UNIT         =          20,
        ERROR_NOT_READY        =          21,
        ERROR_WRITE_FAULT      =          29,
        ERROR_READ_FAULT       =          30,
        ERROR_SHARING_VIOLATION =         32,
        ERROR_LOCK_VIOLATION    =         33,
        ERROR_HANDLE_DISK_FULL  =         39,
        ERROR_BAD_NETPATH       =         53,
        ERROR_NETWORK_BUSY      =         54,
        ERROR_DEV_NOT_EXIST     =         55,
        ERROR_FILE_EXISTS       =         80,
        ERROR_OPEN_FAILED       =         110,
        ERROR_INVALID_NAME      =         123,
        ERROR_BAD_PATHNAME      =         161,
        ERROR_ALREADY_EXISTS    =         183,
    }
end

local function two_arguments (f1,f2)
    return utils.quote_arg(f1)..' '..utils.quote_arg(f2)
end

local function file_op (is_copy,src,dest,flag)
    if flag == 1 and path.exists(dest) then
        return false,"cannot overwrite destination"
    end
    if is_windows then
        -- if we haven't tried to load Alien/LuaJIT FFI before, then do so
        find_ffi_copyfile()
        -- fallback if there's no Alien, just use DOS commands *shudder*
        -- 'rename' involves a copy and then deleting the source.
        if not CopyFile then
            if path.is_windows then
                src = src:gsub("/","\\")
                dest = dest:gsub("/","\\")
            end
            local res, err = execute_command('copy',two_arguments(src,dest))
            if not res then return false,err end
            if not is_copy then
                return execute_command('del',utils.quote_arg(src))
            end
            return true
        else
            if path.isdir(dest) then
                dest = path.join(dest,path.basename(src))
            end
            local ret
            if is_copy then ret = CopyFile(src,dest,flag)
            else ret = MoveFile(src,dest) end
            if ret == 0 then
                local err = GetLastError()
                for name,value in pairs(win32_errors) do
                    if value == err then return false,name end
                end
                return false,"Error #"..err
            else return true
            end
        end
    else -- for Unix, just use cp for now
        return execute_command(is_copy and 'cp' or 'mv',
            two_arguments(src,dest))
    end
end

--- copy a file.
-- @string src source file
-- @string dest destination file or directory
-- @bool flag true if you want to force the copy (default)
-- @treturn bool operation succeeded
-- @raise src and dest must be strings
function dir.copyfile (src,dest,flag)
    assert_string(1,src)
    assert_string(2,dest)
    flag = flag==nil or flag
    return file_op(true,src,dest,flag and 0 or 1)
end

--- move a file.
-- @string src source file
-- @string dest destination file or directory
-- @treturn bool operation succeeded
-- @raise src and dest must be strings
function dir.movefile (src,dest)
    assert_string(1,src)
    assert_string(2,dest)
    return file_op(false,src,dest,0)
end

local function _dirfiles(dirname,attrib)
    local dirs = {}
    local files = {}
    for f in ldir(dirname) do
        if f ~= '.' and f ~= '..' then
            local p = path.join(dirname,f)
            local mode = attrib(p,'mode')
            if mode=='directory' then
                append(dirs,f)
            else
                append(files,f)
            end
        end
    end
    return makelist(dirs), makelist(files)
end


--- return an iterator which walks through a directory tree starting at root.
-- The iterator returns (root,dirs,files)
-- Note that dirs and files are lists of names (i.e. you must say path.join(root,d)
-- to get the actual full path)
-- If bottom_up is false (or not present), then the entries at the current level are returned
-- before we go deeper. This means that you can modify the returned list of directories before
-- continuing.
-- This is a clone of os.walk from the Python libraries.
-- @string root A starting directory
-- @bool bottom_up False if we start listing entries immediately.
-- @bool follow_links follow symbolic links
-- @return an iterator returning root,dirs,files
-- @raise root must be a directory
function dir.walk(root,bottom_up,follow_links)
    assert_dir(1,root)
    local attrib
    if path.is_windows or not follow_links then
        attrib = path.attrib
    else
        attrib = path.link_attrib
    end

    local to_scan = { root }
    local to_return = {}
    local iter = function()
        while #to_scan > 0 do
            local current_root = table.remove(to_scan)
            local dirs,files = _dirfiles(current_root, attrib)
            for _, d in ipairs(dirs) do
                table.insert(to_scan, current_root..path.sep..d)
            end
            if not bottom_up then
                return current_root, dirs, files
            else
                table.insert(to_return, { current_root, dirs, files })
            end
        end
        if #to_return > 0 then
            return utils.unpack(table.remove(to_return))
        end
    end

    return iter
end

--- remove a whole directory tree.
-- Symlinks in the tree will be deleted without following them.
-- @string fullpath A directory path (must be an actual directory, not a symlink)
-- @return true or nil
-- @return error if failed
-- @raise fullpath must be a string
function dir.rmtree(fullpath)
    assert_dir(1,fullpath)
    if path.islink(fullpath) then return false,'will not follow symlink' end
    for root,dirs,files in dir.walk(fullpath,true) do
        if path.islink(root) then
            -- sub dir is a link, remove link, do not follow
            if is_windows then
                -- Windows requires using "rmdir". Deleting the link like a file
                -- will instead delete all files from the target directory!!
                local res, err = rmdir(root)
                if not res then return nil,err .. ": " .. root end
            else
                local res, err = remove(root)
                if not res then return nil,err .. ": " .. root end
            end
        else
            for i,f in ipairs(files) do
                local res, err = remove(path.join(root,f))
                if not res then return nil,err .. ": " .. path.join(root,f) end
            end
            local res, err = rmdir(root)
            if not res then return nil,err .. ": " .. root end
        end
    end
    return true
end


do
  local dirpat
  if path.is_windows then
      dirpat = '(.+)\\[^\\]+$'
  else
      dirpat = '(.+)/[^/]+$'
  end

  local _makepath
  function _makepath(p)
      -- windows root drive case
      if p:find '^%a:[\\]*$' then
          return true
      end
      if not path.isdir(p) then
          local subp = p:match(dirpat)
          if subp then
            local ok, err = _makepath(subp)
            if not ok then return nil, err end
          end
          return mkdir(p)
      else
          return true
      end
  end

  --- create a directory path.
  -- This will create subdirectories as necessary!
  -- @string p A directory path
  -- @return true on success, nil + errormsg on failure
  -- @raise failure to create
  function dir.makepath (p)
      assert_string(1,p)
      if path.is_windows then
          p = p:gsub("/", "\\")
      end
      return _makepath(path.abspath(p))
  end
end

--- clone a directory tree. Will always try to create a new directory structure
-- if necessary.
-- @string path1 the base path of the source tree
-- @string path2 the new base path for the destination
-- @func file_fun an optional function to apply on all files
-- @bool verbose an optional boolean to control the verbosity of the output.
--  It can also be a logging function that behaves like print()
-- @return true, or nil
-- @return error message, or list of failed directory creations
-- @return list of failed file operations
-- @raise path1 and path2 must be strings
-- @usage clonetree('.','../backup',copyfile)
function dir.clonetree (path1,path2,file_fun,verbose)
    assert_string(1,path1)
    assert_string(2,path2)
    if verbose == true then verbose = print end
    local abspath,normcase,isdir,join = path.abspath,path.normcase,path.isdir,path.join
    local faildirs,failfiles = {},{}
    if not isdir(path1) then return raise 'source is not a valid directory' end
    path1 = abspath(normcase(path1))
    path2 = abspath(normcase(path2))
    if verbose then verbose('normalized:',path1,path2) end
    -- particularly NB that the new path isn't fully contained in the old path
    if path1 == path2 then return raise "paths are the same" end
    local _,i2 = path2:find(path1,1,true)
    if i2 == #path1 and path2:sub(i2+1,i2+1) == path.sep then
        return raise 'destination is a subdirectory of the source'
    end
    local cp = path.common_prefix (path1,path2)
    local idx = #cp
    if idx == 0 then -- no common path, but watch out for Windows paths!
        if path1:sub(2,2) == ':' then idx = 3 end
    end
    for root,dirs,files in dir.walk(path1) do
        local opath = path2..root:sub(idx)
        if verbose then verbose('paths:',opath,root) end
        if not isdir(opath) then
            local ret = dir.makepath(opath)
            if not ret then append(faildirs,opath) end
            if verbose then verbose('creating:',opath,ret) end
        end
        if file_fun then
            for i,f in ipairs(files) do
                local p1 = join(root,f)
                local p2 = join(opath,f)
                local ret = file_fun(p1,p2)
                if not ret then append(failfiles,p2) end
                if verbose then
                    verbose('files:',p1,p2,ret)
                end
            end
        end
    end
    return true,faildirs,failfiles
end


-- each entry of the stack is an array with three items:
-- 1. the name of the directory
-- 2. the lfs iterator function
-- 3. the lfs iterator userdata
local function treeiter(iterstack)
    local diriter = iterstack[#iterstack]
    if not diriter then
      return -- done
    end

    local dirname = diriter[1]
    local entry = diriter[2](diriter[3])
    if not entry then
      table.remove(iterstack)
      return treeiter(iterstack) -- tail-call to try next
    end

    if entry ~= "." and entry ~= ".." then
        entry = dirname .. sep .. entry
        if exists(entry) then  -- Just in case a symlink is broken.
            local is_dir = isdir(entry)
            if is_dir then
                table.insert(iterstack, { entry, ldir(entry) })
            end
            return entry, is_dir
        end
    end

    return treeiter(iterstack) -- tail-call to try next
end


--- return an iterator over all entries in a directory tree
-- @string d a directory
-- @return an iterator giving pathname and mode (true for dir, false otherwise)
-- @raise d must be a non-empty string
function dir.dirtree( d )
    assert( d and d ~= "", "directory parameter is missing or empty" )

    local last = sub ( d, -1 )
    if last == sep or last == '/' then
        d = sub( d, 1, -2 )
    end

    local iterstack = { {d, ldir(d)} }

    return treeiter, iterstack
end


--- Recursively returns all the file starting at 'path'. It can optionally take a shell pattern and
-- only returns files that match 'shell_pattern'. If a pattern is given it will do a case insensitive search.
-- @string[opt='.'] start_path  A directory.
-- @string[opt='*'] shell_pattern A shell pattern (see `fnmatch`).
-- @treturn List(string) containing all the files found recursively starting at 'path' and filtered by 'shell_pattern'.
-- @raise start_path must be a directory
function dir.getallfiles( start_path, shell_pattern )
    start_path = start_path or '.'
    assert_dir(1,start_path)
    shell_pattern = shell_pattern or "*"

    local files = {}
    local normcase = path.normcase
    for filename, mode in dir.dirtree( start_path ) do
        if not mode then
            local mask = filemask( shell_pattern )
            if normcase(filename):find( mask ) then
                files[#files + 1] = filename
            end
        end
    end

    return makelist(files)
end

return dir
