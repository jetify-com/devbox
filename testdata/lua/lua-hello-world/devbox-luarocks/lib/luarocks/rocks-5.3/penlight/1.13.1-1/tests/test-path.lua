local path = require 'pl.path'
asserteq = require 'pl.test'.asserteq

function quote(s)
    return '"'..s..'"'
end

function print2(s1,s2)
    print(quote(s1),quote(s2))
end

function slash (p)
  return (p:gsub('\\','/'))
end

-- path.currentdir
do
  local cp = path.currentdir()
  path.chdir("docs")
  asserteq(path.currentdir(), cp .. path.sep .. "docs")
  path.chdir("..")
  asserteq(path.currentdir(), cp)
end

-- path.isdir
asserteq( path.isdir( "docs" ), true )
asserteq( path.isdir( "docs/" ), true )
asserteq( path.isdir( "docs/index.html" ), false )
asserteq( path.isdir( path.currentdir() ), true)
asserteq( path.isdir( "c:\\" ), path.is_windows )
asserteq( path.isdir( "c:/" ), path.is_windows )

-- path.isfile
asserteq( path.isfile( "docs" ), false )
asserteq( path.isfile( "docs/index.html" ), true )

-- path.exists
asserteq( path.exists( "docs"), "docs")
asserteq( path.exists( "docs/index.html"), "docs/index.html")


do  -- path.splitpath & path.splitext
  function testpath(pth,p1,p2,p3)
      local dir,rest = path.splitpath(pth)
      local name,ext = path.splitext(rest)
      asserteq(dir,p1)
      asserteq(name,p2)
      asserteq(ext,p3)
  end

  testpath ([[/bonzo/dog_stuff/cat.txt]],[[/bonzo/dog_stuff]],'cat','.txt')
  testpath ([[/bonzo/dog/cat/fred.stuff]],'/bonzo/dog/cat','fred','.stuff')
  testpath ([[../../alice/jones]],'../../alice','jones','')
  testpath ([[alice]],'','alice','')
  testpath ([[/path-to/dog/]],[[/path-to/dog]],'','')

  asserteq({path.splitpath("some/dir/myfile.txt")}, {"some/dir", "myfile.txt"})
  asserteq({path.splitpath("some/dir/")}, {"some/dir", ""})
  asserteq({path.splitpath("some_dir")}, {"", "some_dir"})

  asserteq({path.splitext("/bonzo/dog_stuff/cat.txt")}, {"/bonzo/dog_stuff/cat", ".txt"})
  asserteq({path.splitext("cat.txt")}, {"cat", ".txt"})
  asserteq({path.splitext("cat")}, {"cat", ""})
  asserteq({path.splitext(".txt")}, {"", ".txt"})
  asserteq({path.splitext("")}, {"", ""})
end


-- TODO: path.abspath

-- TODO: path.dirname

-- TODO: path.basename

-- TODO: path.extension


do -- path.isabs
  asserteq(path.isabs("/hello/path"), true)
  asserteq(path.isabs("hello/path"), false)
  asserteq(path.isabs("./hello/path"), false)
  asserteq(path.isabs("../hello/path"), false)
  if path.is_windows then
    asserteq(path.isabs("c:/"), true)
    asserteq(path.isabs("c:/hello/path"), true)
    asserteq(path.isabs("c:"), false)
    asserteq(path.isabs("c:hello/path"), false)
    asserteq(path.isabs("c:./hello/path"), false)
    asserteq(path.isabs("c:../hello/path"), false)
  end
end


do -- path.join
  assert(path.join("somepath",".") == "somepath"..path.sep..".")
  assert(path.join(".","readme.txt") == "."..path.sep.."readme.txt")
  assert(path.join("/a_dir", "abs_path/") == "/a_dir"..path.sep.."abs_path/")
  assert(path.join("a_dir", "/abs_path/") == "/abs_path/")
  assert(path.join("a_dir", "/abs_path/", "/abs_path2/") == "/abs_path2/")
  assert(path.join("a_dir", "/abs_path/", "not_abs_path2/") == "/abs_path/not_abs_path2/")
  assert(path.join("a_dir", "/abs_path/", "not_abs_path2/", "/abs_path3/", "not_abs_path4/") == "/abs_path3/not_abs_path4/")
  assert(path.join("first","second","third") == "first"..path.sep.."second"..path.sep.."third")
  assert(path.join("first","second","") == "first"..path.sep.."second"..path.sep)
  assert(path.join("first","","third") == "first"..path.sep.."third")
  assert(path.join("","second","third") == "second"..path.sep.."third")
  assert(path.join("","") == "")
end


do -- path.normcase
  if path.iswindows then
    asserteq('c:\\hello\\world', 'c:\\hello\\world')
    asserteq('C:\\Hello\\wORLD', 'c:\\hello\\world')
    asserteq('c:/hello/world', 'c:\\hello\\world')
  else
    asserteq('/Hello/wORLD', '/Hello/wORLD')
  end
end


do  -- path.normpath
  local norm = path.normpath
  local p = norm '/a/b'

  asserteq(norm '/a/fred/../b',p)
  asserteq(norm '/a//b',p)

  function testnorm(p1,p2)
      asserteq(norm(p1):gsub('\\','/'), p2)
  end

  testnorm('a/b/..','a')
  testnorm('a/b/../..','.')
  testnorm('a/b/../c/../../d','d')
  testnorm('a/.','a')
  testnorm('a/./','a')
  testnorm('a/b/.././..','.')
  testnorm('../../a/b','../../a/b')
  testnorm('../../a/b/../../','../..')
  testnorm('../../a/b/../c','../../a/c')
  testnorm('./../../a/b/../c','../../a/c')
  testnorm('a/..b', 'a/..b')
  testnorm('./a', 'a')
  testnorm('a/.', 'a')
  testnorm('a/', 'a')
  testnorm('/a', '/a')
  testnorm('', ".")

  if path.is_windows then
      testnorm('C://a', 'C:/a')
      testnorm('C:/../a', 'C:/../a')
      asserteq(norm [[\a\.\b]], p)
      -- UNC paths
      asserteq(norm [[\\bonzo\..\dog]], [[\\dog]])
      asserteq(norm [[\\?\c:\bonzo\dog\.\]], [[\\?\c:\bonzo\dog]])
  else
      testnorm('//a', '//a')
      testnorm('///a', '/a')
  end

  asserteq(norm '1/2/../3/4/../5',norm '1/3/5')
  asserteq(norm '1/hello/../3/hello/../HELLO',norm '1/3/HELLO')
end


do  --  path.relpath
  local testpath = '/a/B/c'

  function try (p,r)
      asserteq(slash(path.relpath(p,testpath)),r)
  end

  try('/a/B/c/one.lua','one.lua')
  try('/a/B/c/bonZO/two.lua','bonZO/two.lua')
  try('/a/B/three.lua','../three.lua')
  try('/a/four.lua','../../four.lua')
  try('one.lua','one.lua')
  try('../two.lua','../two.lua')
end


-- TODO: path.expanduser

-- TODO: path.tmpname


do --  path.common_prefix
  asserteq(slash(path.common_prefix("../anything","../anything/goes")),"../anything")
  asserteq(slash(path.common_prefix("../anything/goes","../anything")),"../anything")
  asserteq(slash(path.common_prefix("../anything/goes","../anything/goes")),"../anything")
  asserteq(slash(path.common_prefix("../anything/","../anything/")),"../anything")
  asserteq(slash(path.common_prefix("../anything","../anything")),"..")
  asserteq(slash(path.common_prefix("/hello/world","/hello/world/filename.doc")),"/hello/world")
  asserteq(slash(path.common_prefix("/hello/filename.doc","/hello/filename.doc")),"/hello")
  if path.is_windows then
    asserteq(path.common_prefix("c:\\hey\\there","c:\\hey"),"c:\\hey")
    asserteq(path.common_prefix("c:/HEy/there","c:/hEy"),"c:\\hEy")  -- normalized separators, original casing
  end
end


-- TODO: path.package_path
