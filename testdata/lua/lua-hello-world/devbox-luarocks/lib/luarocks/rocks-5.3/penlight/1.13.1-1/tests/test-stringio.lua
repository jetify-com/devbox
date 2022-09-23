local stringio = require 'pl.stringio'
local test = require 'pl.test'
local asserteq = test.asserteq
local T = test.tuple

function fprintf(f,fmt,...)
  f:write(fmt:format(...))
end

fs = stringio.create()
for i = 1,100 do
    fs:write('hello','\n','dolly','\n')
end
asserteq(#fs:value(),1200)

fs = stringio.create()
fs:writef("%s %d",'answer',42)  -- note writef() extension method
asserteq(fs:value(),"answer 42")

inf = stringio.open('10 20 30')
asserteq(T(inf:read('*n','*n','*n')),T(10,20,30))

local txt = [[
Some lines
here are they
not for other
english?

]]

inf = stringio.open (txt)
fs = stringio.create()
for l in inf:lines() do
    fs:write(l,'\n')
end
asserteq(txt,fs:value())

inf = stringio.open '1234567890ABCDEF'
asserteq(T(inf:read(3), inf:read(5), inf:read()),T('123','45678','90ABCDEF'))

s = stringio.open 'one\ntwo'
asserteq(s:read() , 'one')
asserteq(s:read() , 'two')
asserteq(s:read() , nil)
s = stringio.open 'one\ntwo'
iter = s:lines()
asserteq(iter() , 'one')
asserteq(iter() , 'two')
asserteq(iter() , nil)
s = stringio.open 'ABC'
iter = s:lines(1)
asserteq(iter() , 'A')
asserteq(iter() , 'B')
asserteq(iter() , 'C')
asserteq(iter() , nil)

s = stringio.open '20 5.2e-2 52.3'
x,y,z = s:read('*n','*n','*n')
out = stringio.create()
fprintf(out,"%5.2f %5.2f %5.2f!",x,y,z)
asserteq(out:value(),"20.00  0.05 52.30!")

s = stringio.open 'one\ntwo\n\n'
iter = s:lines '*L'
asserteq(iter(),'one\n')
asserteq(iter(),'two\n')
asserteq(iter(),'\n')
asserteq(iter(),nil)




