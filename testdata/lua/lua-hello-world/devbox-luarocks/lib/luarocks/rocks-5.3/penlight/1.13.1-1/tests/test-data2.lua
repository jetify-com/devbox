local utils = require 'pl.utils'
local stringio = require 'pl.stringio'
local data = require 'pl.data'
local test = require 'pl.test'

utils.on_error 'quit'

stuff = [[
Department Name,Employee ID,Project,Hours Booked
sales, 1231,overhead,4
sales,1255,overhead,3
engineering,1501,development,5
engineering,1501,maintenance,3
engineering,1433,maintenance,10
]]

t = data.read(stringio.open(stuff))

q = t:select 'Employee_ID,Hours_Booked where Department_Name == "engineering"'

test.asserteq2(1501,5,q())
test.asserteq2(1501,3,q())
test.asserteq2(1433,10,q())
