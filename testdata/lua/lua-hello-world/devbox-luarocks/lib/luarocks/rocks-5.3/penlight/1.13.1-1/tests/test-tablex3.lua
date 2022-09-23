-- tablex.move when the tables are the same
-- and there are overlapping ranges
T = require 'pl.tablex'
asserteq = require 'pl.test'.asserteq

t1 = {1,2,3,4,5,6,7,8,9,10}
t2 = T.copy(t1)
t3 = T.copy(t1)

T.move(t1,t2,4,1,4)
T.move(t3,t3,4,1,4)
asserteq(t1,t3)
