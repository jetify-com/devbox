--- File manipulation functions: reading, writing, moving and copying.
--
-- This module wraps a number of functions from other modules into a
-- file related module for convenience.
--
-- Dependencies: `pl.utils`, `pl.dir`, `pl.path`
-- @module pl.file
local os = os
local utils = require 'pl.utils'
local dir = require 'pl.dir'
local path = require 'pl.path'

local file = {}

--- return the contents of a file as a string.
-- This function is a copy of `utils.readfile`.
-- @function file.read
file.read = utils.readfile

--- write a string to a file.
-- This function is a copy of `utils.writefile`.
-- @function file.write
file.write = utils.writefile

--- copy a file.
-- This function is a copy of `dir.copyfile`.
-- @function file.copy
file.copy = dir.copyfile

--- move a file.
-- This function is a copy of `dir.movefile`.
-- @function file.move
file.move = dir.movefile

--- Return the time of last access as the number of seconds since the epoch.
-- This function is a copy of `path.getatime`.
-- @function file.access_time
file.access_time = path.getatime

---Return when the file was created.
-- This function is a copy of `path.getctime`.
-- @function file.creation_time
file.creation_time = path.getctime

--- Return the time of last modification.
-- This function is a copy of `path.getmtime`.
-- @function file.modified_time
file.modified_time = path.getmtime

--- Delete a file.
-- This function is a copy of `os.remove`.
-- @function file.delete
file.delete = os.remove

return file
