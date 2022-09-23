--- XML LOM Utilities.
--
-- This implements some useful things on [LOM](http://matthewwild.co.uk/projects/luaexpat/lom.html) documents, such as returned by `lxp.lom.parse`.
-- In particular, it can convert LOM back into XML text, with optional pretty-printing control.
-- It is based on stanza.lua from [Prosody](http://hg.prosody.im/trunk/file/4621c92d2368/util/stanza.lua)
--
--     > d = xml.parse "<nodes><node id='1'>alice</node></nodes>"
--     > = d
--     <nodes><node id='1'>alice</node></nodes>
--     > = xml.tostring(d,'','  ')
--     <nodes>
--        <node id='1'>alice</node>
--     </nodes>
--
-- Can be used as a lightweight one-stop-shop for simple XML processing; a simple XML parser is included
-- but the default is to use `lxp.lom` if it can be found.
-- <pre>
-- Prosody IM
-- Copyright (C) 2008-2010 Matthew Wild
-- Copyright (C) 2008-2010 Waqas Hussain--
-- classic Lua XML parser by Roberto Ierusalimschy.
-- modified to output LOM format.
-- http://lua-users.org/wiki/LuaXml
-- </pre>
-- See @{06-data.md.XML|the Guide}
--
-- Dependencies: `pl.utils`
--
-- Soft Dependencies: `lxp.lom` (fallback is to use basic Lua parser)
-- @module pl.xml

local utils = require 'pl.utils'
local split         =   utils.split
local t_insert      =  table.insert
local t_concat      =  table.concat
local t_remove      =  table.remove
local s_match       =  string.match
local tostring      =      tostring
local setmetatable  =  setmetatable
local getmetatable  =  getmetatable
local pairs         =         pairs
local ipairs        =        ipairs
local type          =          type
local next          =          next
local print         =         print
local unpack        =  utils.unpack
local s_gsub        =   string.gsub
local s_sub         =    string.sub
local s_find        =   string.find
local pcall         =         pcall
local require       =       require


utils.raise_deprecation {
  source = "Penlight " .. utils._VERSION,
  message = "the contents of module 'pl.xml' has been deprecated, please use a more specialized library instead",
  version_removed = "2.0.0",
  deprecated_after = "1.11.0",
  no_trace = true,
}



local _M = {}
local Doc = { __type = "doc" };
Doc.__index = Doc;


local function is_text(s) return type(s) == 'string' end
local function is_tag(d) return type(d) == 'table' and is_text(d.tag) end



--- create a new document node.
-- @tparam string tag the tag name
-- @tparam[opt={}] table attr attributes (table of name-value pairs)
-- @return the Node object
-- @see xml.elem
-- @usage
-- local doc = xml.new("main", { hello = "world", answer = "42" })
-- print(doc)  -->  <main hello='world' answer='42'/>
function _M.new(tag, attr)
  if type(tag) ~= "string" then
    error("expected 'tag' to be a string value, got: " .. type(tag), 2)
  end
  attr = attr or {}
  if type(attr) ~= "table" then
    error("expected 'attr' to be a table value, got: " .. type(attr), 2)
  end

  local doc = { tag = tag, attr = attr, last_add = {}};
  return setmetatable(doc, Doc);
end


--- parse an XML document. By default, this uses lxp.lom.parse, but
-- falls back to basic_parse, or if `use_basic` is truthy
-- @param text_or_filename  file or string representation
-- @param is_file whether text_or_file is a file name or not
-- @param use_basic do a basic parse
-- @return a parsed LOM document with the document metatatables set
-- @return nil, error the error can either be a file error or a parse error
function _M.parse(text_or_filename, is_file, use_basic)
  local parser,status,lom
  if use_basic then
    parser = _M.basic_parse
  else
    status,lom = pcall(require,'lxp.lom')
    if not status then
      parser = _M.basic_parse
    else
      parser = lom.parse
    end
  end

  if is_file then
    local text, err = utils.readfile(text_or_filename)
    if not text then
      return nil, err
    end
    text_or_filename = text
  end

  local doc, err = parser(text_or_filename)
  if not doc then
    return nil, err
  end

  if lom then
    _M.walk(doc, false, function(_, d)
      setmetatable(d, Doc)
    end)
  end
  return doc
end


--- Create a Node with a set of children (text or Nodes) and attributes.
-- @tparam string tag a tag name
-- @tparam table|string items either a single child (text or Node), or a table where the hash
-- part is the attributes and the list part is the children (text or Nodes).
-- @return the new Node
-- @see xml.new
-- @see xml.tags
-- @usage
-- local doc = xml.elem("top", "hello world")                -- <top>hello world</top>
-- local doc = xml.elem("main", xml.new("child"))            -- <main><child/></main>
-- local doc = xml.elem("main", { "this ", "is ", "nice" })  -- <main>this is nice</main>
-- local doc = xml.elem("main", { xml.new "this",
--                                xml.new "is",
--                                xml.new "nice" })          -- <main><this/><is/><nice/></main>
-- local doc = xml.elem("main", { hello = "world" })         -- <main hello='world'/>
-- local doc = xml.elem("main", {
--   "prefix",
--   xml.elem("child", { "this ", "is ", "nice"}),
--   "postfix",
--   attrib = "value"
-- })   -- <main attrib='value'>prefix<child>this is nice</child>postfix</main>"
function _M.elem(tag, items)
  local s = _M.new(tag)
  if is_text(items) then items = {items} end
  if is_tag(items) then
    t_insert(s,items)
  elseif type(items) == 'table' then
    for k,v in pairs(items) do
      if is_text(k) then
        s.attr[k] = v
        t_insert(s.attr,k)
      else
        s[k] = v
      end
    end
  end
  return s
end


--- given a list of names, return a number of element constructors.
-- If passing a comma-separated string, then whitespace surrounding the values
-- will be stripped.
--
-- The returned constructor functions are a shortcut to `xml.elem` where you
-- no longer provide the tag-name, but only the `items` table.
-- @tparam string|table list a list of names, or a comma-separated string.
-- @return (multiple) constructor functions; `function(items)`. For the `items`
-- parameter see `xml.elem`.
-- @see xml.elem
-- @usage
-- local new_parent, new_child = xml.tags 'mom, kid'
-- doc = new_parent {new_child 'Bob', new_child 'Annie'}
-- -- <mom><kid>Bob</kid><kid>Annie</kid></mom>
function _M.tags(list)
  local ctors = {}
  if is_text(list) then
    list = split(list:match("^%s*(.-)%s*$"),'%s*,%s*')
  end
  for i,tag in ipairs(list) do
    local function ctor(items)
      return _M.elem(tag,items)
    end
    ctors[i] = ctor
  end
  return unpack(ctors)
end


--- Adds a document Node, at current position.
-- This updates the last inserted position to the new Node.
-- @tparam string tag the tag name
-- @tparam[opt={}] table attrs attributes (table of name-value pairs)
-- @return the current node (`self`)
-- @usage
-- local doc = xml.new("main")
-- doc:addtag("penlight", { hello = "world"})
-- doc:addtag("expat")  -- added to 'penlight' since position moved
-- print(doc)  -->  <main><penlight hello='world'><expat/></penlight></main>
function Doc:addtag(tag, attrs)
  local s = _M.new(tag, attrs)
  self:add_child(s)
  t_insert(self.last_add, s)
  return self
end


--- Adds a text node, at current position.
-- @tparam string text a string
-- @return the current node (`self`)
-- @usage
-- local doc = xml.new("main")
-- doc:text("penlight")
-- doc:text("expat")
-- print(doc)  -->  <main><penlightexpat</main>
function Doc:text(text)
  self:add_child(text)
  return self
end


--- Moves current position up one level.
-- @return the current node (`self`)
function Doc:up()
  t_remove(self.last_add)
  return self
end


--- Resets current position to top level.
-- Resets to the `self` node.
-- @return the current node (`self`)
function Doc:reset()
  local last_add = self.last_add
  for i = 1,#last_add do
    last_add[i] = nil
  end
  return self
end


--- Append a child to the currrent Node (ignoring current position).
-- @param child a child node (either text or a document)
-- @return the current node (`self`)
-- @usage
-- local doc = xml.new("main")
-- doc:add_direct_child("dog")
-- doc:add_direct_child(xml.new("child"))
-- doc:add_direct_child("cat")
-- print(doc)  -->  <main>dog<child/>cat</main>
function Doc:add_direct_child(child)
  t_insert(self, child)
  return self
end


--- Append a child at the current position (without changing position).
-- @param child a child node (either text or a document)
-- @return the current node (`self`)
-- @usage
-- local doc = xml.new("main")
-- doc:addtag("one")
-- doc:add_child(xml.new("item1"))
-- doc:add_child(xml.new("item2"))
-- doc:add_child(xml.new("item3"))
-- print(doc)  -->  <main><one><item1/><item2/><item3/></one></main>
function Doc:add_child(child)
  (self.last_add[#self.last_add] or self):add_direct_child(child)
  return self
end


--accessing attributes: useful not to have to expose implementation (attr)
--but also can allow attr to be nil in any future optimizations


--- Set attributes of a document node.
-- Will add/overwite values, but will not remove existing ones.
-- Operates on the Node itself, will not take position into account.
-- @tparam table t a table containing attribute/value pairs
-- @return the current node (`self`)
function Doc:set_attribs(t)
  -- TODO: keep array part in sync
  for k,v in pairs(t) do
    self.attr[k] = v
  end
  return self
end


--- Set a single attribute of a document node.
-- Operates on the Node itself, will not take position into account.
-- @param a attribute
-- @param v its value, pass in `nil` to delete the attribute
-- @return the current node (`self`)
function Doc:set_attrib(a,v)
  -- TODO: keep array part in sync
  self.attr[a] = v
  return self
end


--- Gets the attributes of a document node.
-- Operates on the Node itself, will not take position into account.
-- @return table with attributes (attribute/value pairs)
function Doc:get_attribs()
  return self.attr
end



local template_cache do
  local templ_cache = {}

  -- @param templ a template, a string being valid xml to be parsed, or a Node object
  function template_cache(templ)
    if is_text(templ) then
      if templ_cache[templ] then
        -- cache hit
        return templ_cache[templ]

      else
        -- parse and cache
        local ptempl, err = _M.parse(templ,false,true)
        if not ptempl then
          return nil, err
        end
        templ_cache[templ] = ptempl
        return ptempl
      end
    end

    if is_tag(templ) then
      return templ
    end

    return nil, "template is not a document"
  end
end


do
  local function is_data(data)
    return #data == 0 or type(data[1]) ~= 'table'
  end


  local function prepare_data(data)
    -- a hack for ensuring that $1 maps to first element of data, etc.
    -- Either this or could change the gsub call just below.
    for i,v in ipairs(data) do
      data[tostring(i)] = v
    end
  end

  --- create a substituted copy of a document,
  -- @param template may be a document or a string representation which will be parsed and cached
  -- @param data a table of name-value pairs or a list of such tables
  -- @return an XML document
  function Doc.subst(template, data)
    if type(data) ~= 'table' or not next(data) then
      return nil, "data must be a non-empty table"
    end

    if is_data(data) then
      prepare_data(data)
    end

    local templ, err = template_cache(template)
    if err then
      return nil, err
    end

    local function _subst(item)
      return _M.clone(templ, function(s)
        return s:gsub('%$(%w+)', item)
      end)
    end

    if is_data(data) then
      return _subst(data)
    end

    local list = {}
    for _, item in ipairs(data) do
      prepare_data(item)
      t_insert(list, _subst(item))
    end

    if data.tag then
      list = _M.elem(data.tag,list)
    end
    return list
  end
end


--- Return the first child with a given tag name (non-recursive).
-- @param tag the tag name
-- @return the child Node found or `nil` if not found
function Doc:child_with_name(tag)
  for _, child in ipairs(self) do
    if child.tag == tag then
      return child
    end
  end
end


do
  -- @param self document node to traverse
  -- @param tag tag-name to look for
  -- @param list array table to add the matching ones to
  -- @param recurse if truthy, recursivly search the node
  local function _children_with_name(self, tag, list, recurse)
    -- TODO: protect against recursion
    for _, child in ipairs(self) do
      if type(child) == 'table' then
        if child.tag == tag then
          t_insert(list, child)
        end
        if recurse then
          _children_with_name(child, tag, list, recurse)
        end
      end
    end
  end

  --- Returns all elements in a document that have a given tag.
  -- @tparam string tag a tag name
  -- @tparam[opt=false] boolean dont_recurse optionally only return the immediate children with this tag name
  -- @return a list of elements found, list will be empty if none was found.
  function Doc:get_elements_with_name(tag, dont_recurse)
    local res = {}
    _children_with_name(self, tag, res, not dont_recurse)
    return res
  end
end



--- Iterator over all children of a document node, including text nodes.
-- This function is not recursive, so returns only direct child nodes.
-- @return iterator that returns a single Node per iteration.
function Doc:children()
  local i = 0;
  return function (a)
    i = i + 1
    return a[i];
  end, self, i;
end


--- Return the first child element of a node, if it exists.
-- This will skip text nodes.
-- @return first child Node or `nil` if there is none.
function Doc:first_childtag()
  if #self == 0 then
    return
  end
  for _, t in ipairs(self) do
    if is_tag(t) then
      return t
    end
  end
end


--- Iterator that matches tag names, and a namespace (non-recursive).
-- @tparam[opt=nil] string tag tag names to return. Returns all tags if not provided.
-- @tparam[opt=nil] string xmlns the namespace value ('xmlns' attribute) to return. If not
-- provided will match all namespaces.
-- @return iterator that returns a single Node per iteration.
function Doc:matching_tags(tag, xmlns)
  -- TODO: this doesn't make sense??? namespaces are not "xmnls", as matched below
  -- but "xmlns:name"... so should be a string-prefix match if anything...
  xmlns = xmlns or self.attr.xmlns;
  local tags = self
  local next_i = 1
  local max_i = #tags
  local node
  return function ()
      for i = next_i, max_i do
        node = tags[i];
        if (not tag or node.tag == tag) and
           (not xmlns or xmlns == node.attr.xmlns) then
          next_i = i + 1
          return node
        end
      end
    end, tags, next_i
end


--- Iterator over all child tags of a document node. This will skip over
-- text nodes.
-- @return iterator that returns a single Node per iteration.
function Doc:childtags()
  local i = 0;
  return function (a)
    local v
      repeat
        i = i + 1
        v = self[i]
        if v and type(v) == 'table' then
          return v
        end
      until not v
    end, self[1], i;
end


--- Visit child Nodes of a node and call a function, possibly modifying the document.
-- Text elements will be skipped.
-- This is not recursive, so only direct children will be passed.
-- @tparam function callback a function with signature `function(node)`, passed the node.
-- The element will be updated with the returned value, or deleted if it returns `nil`.
function Doc:maptags(callback)
  local i = 1;

  while i <= #self do
    if is_tag(self[i]) then
      local ret = callback(self[i]);
      if ret == nil then
        -- remove it
        t_remove(self, i);

      else
        -- update it
        self[i] = ret;
        i = i + 1;
      end
    else
      i = i + 1
    end
  end

  return self;
end


do
  local escape_table = {
    ["'"] = "&apos;",
    ['"'] = "&quot;",
    ["<"] = "&lt;",
    [">"] = "&gt;",
    ["&"] = "&amp;",
  }

  --- Escapes a string for safe use in xml.
  -- Handles quotes(single+double), less-than, greater-than, and ampersand.
  -- @tparam string str string value to escape
  -- @return escaped string
  -- @usage
  -- local esc = xml.xml_escape([["'<>&]])  --> "&quot;&apos;&lt;&gt;&amp;"
  function _M.xml_escape(str)
    return (s_gsub(str, "['&<>\"]", escape_table))
  end
end
local xml_escape = _M.xml_escape

do
  local escape_table = {
    quot = '"',
    apos = "'",
    lt = "<",
    gt = ">",
    amp = "&",
  }

  --- Unescapes a string from xml.
  -- Handles quotes(single+double), less-than, greater-than, and ampersand.
  -- @tparam string str string value to unescape
  -- @return unescaped string
  -- @usage
  -- local unesc = xml.xml_escape("&quot;&apos;&lt;&gt;&amp;")  --> [["'<>&]]
  function _M.xml_unescape(str)
    return (str:gsub( "&(%a+);", escape_table))
  end
end
local xml_unescape = _M.xml_unescape

-- pretty printing
-- if indent, then put each new tag on its own line
-- if attr_indent, put each new attribute on its own line
local function _dostring(t, buf, parentns, block_indent, tag_indent, attr_indent)
  local nsid = 0
  local tag = t.tag

  local lf = ""
  if tag_indent then
    lf = '\n'..block_indent
  end

  local alf = " "
  if attr_indent then
    alf = '\n'..block_indent..attr_indent
  end

  t_insert(buf, lf.."<"..tag)

  local function write_attr(k,v)
    if s_find(k, "\1", 1, true) then
      nsid = nsid + 1
      local ns, attrk = s_match(k, "^([^\1]*)\1?(.*)$")
      t_insert(buf, " xmlns:ns"..nsid.."='"..xml_escape(ns).."' ".."ns"..nsid..":"..attrk.."='"..xml_escape(v).."'")

    elseif not (k == "xmlns" and v == parentns) then
      t_insert(buf, alf..k.."='"..xml_escape(v).."'");
    end
  end

  -- it's useful for testing to have predictable attribute ordering, if available
  if #t.attr > 0 then
    -- TODO: the key-value list is leading, what if they are not in-sync
    for _,k in ipairs(t.attr) do
      write_attr(k,t.attr[k])
    end
  else
    for k, v in pairs(t.attr) do
      write_attr(k,v)
    end
  end

  local len = #t
  local has_children

  if len == 0 then
    t_insert(buf, attr_indent and '\n'..block_indent.."/>" or "/>")

  else
    t_insert(buf, ">");

    for n = 1, len do
      local child = t[n]

      if child.tag then
        has_children = true
        _dostring(child, buf, t.attr.xmlns, block_indent and block_indent..tag_indent, tag_indent, attr_indent)

      else
        -- text element
        t_insert(buf, xml_escape(child))
      end
    end

    t_insert(buf, (has_children and lf or '').."</"..tag..">");
  end
end

--- Function to pretty-print an XML document.
-- @param doc an XML document
-- @tparam[opt] string|int b_ind an initial block-indent (required when `t_ind` is set)
-- @tparam[opt] string|int t_ind an tag-indent for each level (required when `a_ind` is set)
-- @tparam[opt] string|int a_ind if given, indent each attribute pair and put on a separate line
-- @tparam[opt] string|bool xml_preface force prefacing with default or custom <?xml...>, if truthy then `&lt;?xml version='1.0'?&gt;` will be used as default.
-- @return a string representation
-- @see Doc:tostring
function _M.tostring(doc, b_ind, t_ind, a_ind, xml_preface)
  local buf = {}

  if type(b_ind) == "number" then b_ind = (" "):rep(b_ind) end
  if type(t_ind) == "number" then t_ind = (" "):rep(t_ind) end
  if type(a_ind) == "number" then a_ind = (" "):rep(a_ind) end

  if xml_preface then
    if type(xml_preface) == "string" then
      buf[1] = xml_preface
    else
      buf[1] = "<?xml version='1.0'?>"
    end
  end

  _dostring(doc, buf, nil, b_ind, t_ind, a_ind, xml_preface)

  return t_concat(buf)
end


Doc.__tostring = _M.tostring


--- Method to pretty-print an XML document.
-- Invokes `xml.tostring`.
-- @tparam[opt] string|int b_ind an initial indent (required when `t_ind` is set)
-- @tparam[opt] string|int t_ind an indent for each level (required when `a_ind` is set)
-- @tparam[opt] string|int a_ind if given, indent each attribute pair and put on a separate line
-- @tparam[opt="&lt;?xml version='1.0'?&gt;"] string xml_preface force prefacing with default or custom <?xml...>
-- @return a string representation
-- @see xml.tostring
function Doc:tostring(b_ind, t_ind, a_ind, xml_preface)
  return _M.tostring(self, b_ind, t_ind, a_ind, xml_preface)
end


--- get the full text value of an element.
-- @return a single string with all text elements concatenated
-- @usage
-- local doc = xml.new("main")
-- doc:text("one")
-- doc:add_child(xml.elem "two")
-- doc:text("three")
--
-- local t = doc:get_text()    -->  "onethree"
function Doc:get_text()
  local res = {}
  for i,el in ipairs(self) do
    if is_text(el) then t_insert(res,el) end
  end
  return t_concat(res);
end


do
  local function _copy(object, kind, parent, strsubst, lookup_table)
    if type(object) ~= "table" then
      if strsubst and is_text(object) then
        return strsubst(object, kind, parent)
      else
        return object
      end
    end

    if lookup_table[object] then
      error("recursion detected")
    end
    lookup_table[object] = true

    local new_table = {}
    lookup_table[object] = new_table

    local tag = object.tag
    new_table.tag = _copy(tag, '*TAG', parent, strsubst, lookup_table)

    if object.attr then
      local res = {}
      for attr, value in pairs(object.attr) do
        if type(attr) == "string" then
          res[attr] = _copy(value, attr, object, strsubst, lookup_table)
        end
      end
      new_table.attr = res
    end

    for index = 1, #object do
      local v = _copy(object[index], '*TEXT', object, strsubst, lookup_table)
      t_insert(new_table,v)
    end

    return setmetatable(new_table, getmetatable(object))
  end

  --- Returns a copy of a document.
  -- The `strsubst` parameter is a callback with signature `function(object, kind, parent)`.
  --
  -- Param `kind` has the following values, and parameters:
  --
  -- - `"*TAG"`: `object` is the tag-name, `parent` is the Node object. Returns the new tag name.
  --
  -- - `"*TEXT"`: `object` is the text-element, `parent` is the Node object. Returns the new text value.
  --
  -- - other strings not prefixed with `*`: `kind` is the attribute name, `object` is the
  --   attribute value, `parent` is the Node object. Returns the new attribute value.
  --
  -- @tparam Node|string doc a Node object or string (text node)
  -- @tparam[opt] function strsubst an optional function for handling string copying
  -- which could do substitution, etc.
  -- @return copy of the document
  -- @see Doc:filter
  function _M.clone(doc, strsubst)
    return _copy(doc, nil, nil, strsubst, {})
  end
end


--- Returns a copy of a document.
-- This is the method version of `xml.clone`.
-- @see xml.clone
-- @name Doc:filter
-- @tparam[opt] function strsubst an optional function for handling string copying
Doc.filter = _M.clone -- also available as method

do
  local function _compare(t1, t2, recurse_check)

    local ty1 = type(t1)
    local ty2 = type(t2)

    if ty1 ~= ty2 then
      return false, 'type mismatch'
    end

    if ty1 == 'string' then
      if t1 == t2 then
        return true
      else
        return false, 'text '..t1..' ~= text '..t2
      end
    end

    if ty1 ~= 'table' or ty2 ~= 'table' then
      return false, 'not a document'
    end

    if recurse_check[t1] then
      return false, "recursive document"
    end
    recurse_check[t1] = true

    if t1.tag ~= t2.tag then
      return false, 'tag  '..t1.tag..' ~= tag '..t2.tag
    end

    if #t1 ~= #t2 then
      return false, 'size '..#t1..' ~= size '..#t2..' for tag '..t1.tag
    end

    -- compare attributes
    for k,v in pairs(t1.attr) do
      local t2_value = t2.attr[k]
      if type(k) == "string" then
        if t2_value ~= v then return false, 'mismatch attrib' end
      else
        if t2_value ~= nil and t2_value ~= v then return false, "mismatch attrib order" end
      end
    end
    for k,v in pairs(t2.attr) do
      local t1_value = t1.attr[k]
      if type(k) == "string" then
        if t1_value ~= v then return false, 'mismatch attrib' end
      else
        if t1_value ~= nil and t1_value ~= v then return false, "mismatch attrib order" end
      end
    end

    -- compare children
    for i = 1, #t1 do
      local ok, err = _compare(t1[i], t2[i], recurse_check)
      if not ok then
        return ok, err
      end
    end
    return true
  end

  --- Compare two documents or elements.
  -- Equality is based on tag, child nodes (text and tags), attributes and order
  -- of those (order only fails if both are given, and not equal).
  -- @tparam Node|string t1 a Node object or string (text node)
  -- @tparam Node|string t2 a Node object or string (text node)
  -- @treturn boolean `true` when the Nodes are equal.
  function _M.compare(t1,t2)
    return _compare(t1, t2, {})
  end
end


--- is this value a document element?
-- @param d any value
-- @treturn boolean `true` if it is a `table` with property `tag` being a string value.
-- @name is_tag
_M.is_tag = is_tag


do
  local function _walk(doc, depth_first, operation, recurse_check)
    if not depth_first then operation(doc.tag, doc) end
    for _,d in ipairs(doc) do
      if is_tag(d) then
        assert(not recurse_check[d], "recursion detected")
        recurse_check[d] = true
        _walk(d, depth_first, operation, recurse_check)
      end
    end
    if depth_first then operation(doc.tag, doc) end
  end

  --- Calls a function recursively over Nodes in the document.
  -- Will only call on tags, it will skip text nodes.
  -- The function signature for `operation` is `function(tag_name, Node)`.
  -- @tparam Node|string doc a Node object or string (text node)
  -- @tparam boolean depth_first visit child nodes first, then the current node
  -- @tparam function operation a function which will receive the current tag name and current node.
  function _M.walk(doc, depth_first, operation)
    return _walk(doc, depth_first, operation, {})
  end
end


local html_empty_elements = { --lists all HTML empty (void) elements
    br      = true,
    img     = true,
    meta    = true,
    frame   = true,
    area    = true,
    hr      = true,
    base    = true,
    col     = true,
    link    = true,
    input   = true,
    option  = true,
    param   = true,
    isindex = true,
    embed = true,
}

--- Parse a well-formed HTML file as a string.
-- Tags are case-insenstive, DOCTYPE is ignored, and empty elements can be .. empty.
-- @param s the HTML
function _M.parsehtml(s)
    return _M.basic_parse(s,false,true)
end

--- Parse a simple XML document using a pure Lua parser based on Robero Ierusalimschy's original version.
-- @param s the XML document to be parsed.
-- @param all_text  if true, preserves all whitespace. Otherwise only text containing non-whitespace is included.
-- @param html if true, uses relaxed HTML rules for parsing
function _M.basic_parse(s, all_text, html)
    local stack = {}
    local top = {}

    local function parseargs(s)
      local arg = {}
      s:gsub("([%w:%-_]+)%s*=%s*([\"'])(.-)%2", function (w, _, a)
        if html then w = w:lower() end
        arg[w] = xml_unescape(a)
      end)
      if html then
        s:gsub("([%w:%-_]+)%s*=%s*([^\"']+)%s*", function (w, a)
          w = w:lower()
          arg[w] = xml_unescape(a)
        end)
      end
      return arg
    end

    t_insert(stack, top)
    local ni,c,label,xarg, empty, _, istart
    local i = 1
    local j
    -- we're not interested in <?xml version="1.0"?>
    _,istart = s_find(s,'^%s*<%?[^%?]+%?>%s*')
    if not istart then -- or <!DOCTYPE ...>
        _,istart = s_find(s,'^%s*<!DOCTYPE.->%s*')
    end
    if istart then i = istart+1 end
    while true do
        ni,j,c,label,xarg, empty = s_find(s, "<([%/!]?)([%w:%-_]+)(.-)(%/?)>", i)
        if not ni then break end
        if c == "!" then -- comment
            -- case where there's no space inside comment
            if not (label:match '%-%-$' and xarg == '') then
                if xarg:match '%-%-$' then -- we've grabbed it all
                    j = j - 2
                end
                -- match end of comment
                _,j = s_find(s, "-->", j, true)
            end
        else
            local text = s_sub(s, i, ni-1)
            if html then
                label = label:lower()
                if html_empty_elements[label] then empty = "/" end
            end
            if all_text or not s_find(text, "^%s*$") then
                t_insert(top, xml_unescape(text))
            end
            if empty == "/" then  -- empty element tag
                t_insert(top, setmetatable({tag=label, attr=parseargs(xarg), empty=1},Doc))
            elseif c == "" then   -- start tag
                top = setmetatable({tag=label, attr=parseargs(xarg)},Doc)
                t_insert(stack, top)   -- new level
            else  -- end tag
                local toclose = t_remove(stack)  -- remove top
                top = stack[#stack]
                if #stack < 1 then
                    error("nothing to close with "..label..':'..text)
                end
                if toclose.tag ~= label then
                    error("trying to close "..toclose.tag.." with "..label.." "..text)
                end
                t_insert(top, toclose)
            end
        end
        i = j+1
    end
    local text = s_sub(s, i)
    if all_text or  not s_find(text, "^%s*$") then
        t_insert(stack[#stack], xml_unescape(text))
    end
    if #stack > 1 then
        error("unclosed "..stack[#stack].tag)
    end
    local res = stack[1]
    return is_text(res[1]) and res[2] or res[1]
end

do
  local match do

    local function empty(attr) return not attr or not next(attr) end

    local append_capture do
      -- returns the key,value pair from a table if it has exactly one entry
      local function has_one_element(t)
          local key,value = next(t)
          if next(t,key) ~= nil then return false end
          return key,value
      end

      function append_capture(res,tbl)
          if not empty(tbl) then -- no point in capturing empty tables...
              local key
              if tbl._ then  -- if $_ was set then it is meant as the top-level key for the captured table
                  key = tbl._
                  tbl._ = nil
                  if empty(tbl) then return end
              end
              -- a table with only one pair {[0]=value} shall be reduced to that value
              local numkey,val = has_one_element(tbl)
              if numkey == 0 then tbl = val end
              if key then
                  res[key] = tbl
              else -- otherwise, we append the captured table
                  t_insert(res,tbl)
              end
          end
      end
    end

    local function make_number(pat)
        if pat:find '^%d+$' then -- $1 etc means use this as an array location
            pat = tonumber(pat)
        end
        return pat
    end

    local function capture_attrib(res,pat,value)
        pat = make_number(pat:sub(2))
        res[pat] = value
        return true
    end

    function match(d,pat,res,keep_going)
        local ret = true
        if d == nil then d = '' end --return false end
        -- attribute string matching is straight equality, except if the pattern is a $ capture,
        -- which always succeeds.
        if is_text(d) then
            if not is_text(pat) then return false end
            if _M.debug then print(d,pat) end
            if pat:find '^%$' then
                return capture_attrib(res,pat,d)
            else
                return d == pat
            end
        else
        if _M.debug then print(d.tag,pat.tag) end
            -- this is an element node. For a match to succeed, the attributes must
            -- match as well.
            -- a tagname in the pattern ending with '-' is a wildcard and matches like an attribute
            local tagpat = pat.tag:match '^(.-)%-$'
            if tagpat then
                tagpat = make_number(tagpat)
                res[tagpat] = d.tag
            end
            if d.tag == pat.tag or tagpat then

                if not empty(pat.attr) then
                    if empty(d.attr) then ret =  false
                    else
                        for prop,pval in pairs(pat.attr) do
                            local dval = d.attr[prop]
                            if not match(dval,pval,res) then ret = false;  break end
                        end
                    end
                end
                -- the pattern may have child nodes. We match partially, so that {P1,P2} shall match {X,P1,X,X,P2,..}
                if ret and #pat > 0 then
                    local i,j = 1,1
                    local function next_elem()
                        j = j + 1  -- next child element of data
                        if is_text(d[j]) then j = j + 1 end
                        return j <= #d
                    end
                    repeat
                        local p = pat[i]
                        -- repeated {{<...>}} patterns  shall match one or more elements
                        -- so e.g. {P+} will match {X,X,P,P,X,P,X,X,X}
                        if is_tag(p) and p.repeated then
                            local found
                            repeat
                                local tbl = {}
                                ret = match(d[j],p,tbl,false)
                                if ret then
                                    found = false --true
                                    append_capture(res,tbl)
                                end
                            until not next_elem() or (found and not ret)
                            i = i + 1
                        else
                            ret = match(d[j],p,res,false)
                            if ret then i = i + 1 end
                        end
                    until not next_elem() or i > #pat -- run out of elements or patterns to match
                    -- if every element in our pattern matched ok, then it's been a successful match
                    if i > #pat then return true end
                end
                if ret then return true end
            else
                ret = false
            end
            -- keep going anyway - look at the children!
            if keep_going then
                for child in d:childtags() do
                    ret = match(child,pat,res,keep_going)
                    if ret then break end
                end
            end
        end
        return ret
    end
  end

  --- does something...
  function Doc:match(pat)
      local err
      pat,err = template_cache(pat)
      if not pat then return nil, err end
      _M.walk(pat,false,function(_,d)
          if is_text(d[1]) and is_tag(d[2]) and is_text(d[3]) and
            d[1]:find '%s*{{' and d[3]:find '}}%s*' then
            t_remove(d,1)
            t_remove(d,2)
            d[1].repeated = true
          end
      end)

      local res = {}
      local ret = match(self,pat,res,true)
      return res,ret
  end
end


return _M

