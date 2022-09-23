local data = require 'pl.data'
local List = require 'pl.List'
local array = require 'pl.array2d'
local func = require 'pl.func'
local seq = require 'pl.seq'
local stringio = require 'pl.stringio'
local open = stringio. open
local asserteq = require 'pl.test' . asserteq
local T = require 'pl.test'. tuple

--[=[
dat,err = data.read(open [[
1.0 0.1
0.2 1.3
]])

if err then print(err) end

require 'pl.pretty'.dump(dat)
os.exit(0)
--]=]

-- tab-separated data, explicit column names
local t1f = open [[
EventID	Magnitude	LocationX	LocationY	LocationZ	LocationError	EventDate	DataFile
981124001	2.0	18988.4	10047.1	4149.7	33.8	24/11/1998 11:18:05	981124DF.AAB
981125001	0.8	19104.0	9970.4	5088.7	3.0	25/11/1998 05:44:54	981125DF.AAB
981127003	0.5	19012.5	9946.9	3831.2	46.0	27/11/1998 17:15:17	981127DF.AAD
981127005	0.6	18676.4	10606.2	3761.9	4.4	27/11/1998 17:46:36	981127DF.AAF
981127006	0.2	19109.9	9716.5	3612.0	11.8	27/11/1998 19:29:51	981127DF.AAG
]]

local t1 = data.read (t1f)
-- column_by_name returns a List
asserteq(t1:column_by_name 'Magnitude',List{2,0.8,0.5,0.6,0.2})
-- can use array.column as well
asserteq(array.column(t1,2),{2,0.8,0.5,0.6,0.2})

-- only numerical columns (deduced from first data row) are converted by default
-- can look up indices in the list fieldnames.
local EDI = t1.fieldnames:index 'EventDate'
assert(type(t1[1][EDI]) == 'string')

-- select method returns a sequence, in this case single-valued.
-- (Note that seq.copy returns a List)
asserteq(seq(t1:select 'LocationX where Magnitude > 0.5'):copy(),List{18988.4,19104,18676.4})

--[[
--a common select usage pattern:
for event,mag in t1:select 'EventID,Magnitude sort by Magnitude desc' do
    print(event,mag)
end
--]]

-- space-separated, but with last field containing spaces.
local t2f = open [[
USER PID %MEM %CPU COMMAND
sdonovan 2333  0.3 0.1 background --n=2
root 2332  0.4  0.2 fred --start=yes
root 2338  0.2  0.1 backyard-process
]]

local t2,err = data.read(t2f,{last_field_collect=true})
if not t2 then return print (err) end

-- the last_field_collect option is useful with space-delimited data where the last
-- field may contain spaces. Otherwise, a record count mismatch should be an error!
local lt2 = List(t2[2])
asserteq(lt2:join ',','root,2332,0.4,0.2,fred --start=yes')

-- fieldnames are converted into valid identifiers by substituting _
-- (we do this to make select queries parseable by Lua)
asserteq(t2.fieldnames,List{'USER','PID','_MEM','_CPU','COMMAND'})

-- select queries are NOT SQL so remember to use == ! (and no 'between' operator, sorry)
--s,err = t2:select('_MEM where USER="root"')
--assert(err == [[[string "tmp"]:9: unexpected symbol near '=']])

local s = t2:select('_MEM where USER=="root"')
assert(s() == 0.4)
assert(s() == 0.2)
assert(s() == nil)

-- CSV, Excel style. Double-quoted fields are allowed, and they may contain commas!
local t3f = open [[
"Department Name","Employee ID",Project,"Hours Booked"
sales,1231,overhead,4
sales,1255,overhead,3
engineering,1501,development,5
engineering,1501,maintenance,3
engineering,1433,maintenance,10
]]

local t3 = data.read(t3f,{csv=true})

-- although fieldnames are turned in valid Lua identifiers, there is always `original_fieldnames`
asserteq(t3.fieldnames,List{'Department_Name','Employee_ID','Project','Hours_Booked'})
asserteq(t3.original_fieldnames,List{'Department Name','Employee ID','Project','Hours Booked'})

-- a common operation is to select using a given list of columns, and each row
-- on some explicit condition. The select() method can take a table with these
-- parameters
local keepcols = {'Employee_ID','Hours_Booked'}

local q = t3:select { fields = keepcols,
    where = function(row) return row[1]=='engineering' end
    }

asserteq(seq.copy2(q),{{1501,5},{1501,3},{1433,10}})

-- another pattern is doing a select to restrict rows & columns, process some
-- fields and write out the modified rows.

local outf = stringio.create()

local names = {[1501]='don',[1433]='dilbert'}

t3:write_row (outf,{'Employee','Hours_Booked'})
q = t3:select_row {fields=keepcols,where=func.Eq(func._1[1],'engineering')}
for row in q do
    row[1] = names[row[1]]
    t3:write_row(outf,row)
end

asserteq(outf:value(),
[[
Employee,Hours_Booked
don,5
don,3
dilbert,10
]])

-- data may not always have column headers. When creating a data object
-- from a two-dimensional array, may specify the fieldnames, as a list or a string.
-- The delimiter is deduced from the fieldname string, so a string just containing
-- the delimiter will set it,  and the fieldnames will be empty.
local dat = List()
local row = List.range(1,10)
for i = 1,10 do
    dat:append(row:map('*',i))
end
dat = data.new(dat,',')
local out = stringio.create()
dat:write(out,',')
asserteq(out:value(), [[
1,2,3,4,5,6,7,8,9,10
2,4,6,8,10,12,14,16,18,20
3,6,9,12,15,18,21,24,27,30
4,8,12,16,20,24,28,32,36,40
5,10,15,20,25,30,35,40,45,50
6,12,18,24,30,36,42,48,54,60
7,14,21,28,35,42,49,56,63,70
8,16,24,32,40,48,56,64,72,80
9,18,27,36,45,54,63,72,81,90
10,20,30,40,50,60,70,80,90,100
]])

-- you can always use numerical field indices, AWK-style;
-- note how the copy_select method gives you a data object instead of an
-- iterator over the fields
local res = dat:copy_select '$1,$3 where $1 > 5'
local L = List
asserteq(L(res),L{
    L{6, 18},
    L{7,21},
    L{8,24},
    L{9,27},
    L{10,30},
})

-- the column_by_name method may take a fieldname or an index
asserteq(dat:column_by_name(2), L{2,4,6,8,10,12,14,16,18,20})

-- the field list may contain expressions or even constants
local q = dat:select '$3,2*$4 where $1 == 8'
asserteq(T(q()),T(24,64))

dat,err = data.read(open [[
1.0 0.1
0.2 1.3
]])

if err then print(err) end

-- if a method cannot be found, then we look up in array2d
-- array2d.flatten(t) makes a 1D list out of a 2D array,
-- and then List.minmax() gets the extrema.

asserteq(T(dat:flatten():minmax()),T(0.1,1.3))

local f = open [[
Time Message
1266840760 +# EE7C0600006F0D00C00F06010302054000000308010A00002B00407B00
1266840760 closure data 0.000000 1972 1972 0
1266840760 ++ 1266840760 EE 1
1266840760 +# EE7C0600006F0D00C00F06010302054000000408020A00002B00407B00
1266840764 closure data 0.000000 1972 1972 0
1266840764 ++ 1266840764 EE 1
1266840764 +# EE7C0600006F0D00C00F06010302054000000508030A00002B00407B00
1266840768 duplicate?
1266840768 +# EE7C0600006F0D00C00F06010302054000000508030A00002B00407B00
1266840768 closure data 0.000000 1972 1972 0
]]

-- the `convert` option provides custom converters for each specified column.
-- Here we convert the timestamps into Date objects and collect everything
-- else into one field
local Date = require 'pl.Date'

local function date_convert (ds)
    return Date(tonumber(ds))
end

local d = data.read(f,{convert={[1]=date_convert},last_field_collect=true})

asserteq(#d[1],2)
asserteq(d[2][1]:year(),2010)

d = {{1,2,3},{10,20,30}}
out = stringio.create()
data.write(d,out,{'A','B','C'},',')
asserteq(out:value(),
[[
A,B,C
1,2,3
10,20,30
]])

out = stringio.create()
d.fieldnames = {'A','B','C'}
data.write(d,out)

asserteq(out:value(),
[[
A	B	C
1	2	3
10	20	30
]])


d = data.read(stringio.open 'One,Two\n1,\n,20\n',{csv=true})
asserteq(d,{
    {1,0},{0,20},
    original_fieldnames={"One","Two"},fieldnames={"One","Two"},delim=","
})
