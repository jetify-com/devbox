local xml = require 'pl.xml'
local asserteq = require 'pl.test'.asserteq
local dump = require 'pl.pretty'.dump
local path = require 'pl.path'
local utils = require 'pl.utils'

-- Prosody stanza.lua style XML building

d = xml.new 'top' : addtag 'child' : text 'alice' : up() : addtag 'child' : text 'bob'

d = xml.new 'children' :
  addtag 'child' :
    addtag 'name' :
      text 'alice' :
      up() :
    addtag 'age' :
      text '5' :
      up() :
    addtag('toy',{type='fluffy'}) :
      up() :
    up() :
  addtag 'child':
    addtag 'name' :
      text 'bob' :
      up() :
    addtag 'age' :
      text '6' :
      up() :
    addtag('toy',{type='squeaky'})

asserteq(xml.tostring(d,'','  '), [[

<children>
  <child>
    <name>alice</name>
    <age>5</age>
    <toy type='fluffy'/>
  </child>
  <child>
    <name>bob</name>
    <age>6</age>
    <toy type='squeaky'/>
  </child>
</children>]])

-- Orbit-style 'xmlification'

local children,child,toy,name,age = xml.tags 'children, child, toy, name, age'

d1 = children {
    child {name 'alice', age '5', toy {type='fluffy'}},
    child {name 'bob', age '6', toy {type='squeaky'}}
}
assert(xml.compare(d,d1))

-- or we can use a template document to convert Lua data to LOM

templ = child {name '$name', age '$age', toy{type='$toy'}}

d2 = children(templ:subst{
    {name='alice',age='5',toy='fluffy'},
    {name='bob',age='6',toy='squeaky'}
})

assert(xml.compare(d1,d2))

-- Parsing Google Weather service results --

local joburg = [[
<?xml version="1.0"?>
<xml_api_reply version='1'>
  <weather module_id='0' tab_id='0' mobile_zipped='1' section='0' row='0' mobile_row='0'>
    <forecast_information>
      <city data='Johannesburg, Gauteng'/>
      <postal_code data='Johannesburg,ZA'/>
      <latitude_e6 data=''/>
      <longitude_e6 data=''/>
      <forecast_date data='2010-10-02'/>
      <current_date_time data='2010-10-02 18:30:00 +0000'/>
      <unit_system data='US'/>
    </forecast_information>
    <current_conditions>
      <condition data='Clear'/>
      <temp_f data='75'/>
      <temp_c data='24'/>
      <humidity data='Humidity: 19%'/>
      <icon data='/ig/images/weather/sunny.gif'/>
      <wind_condition data='Wind: NW at 7 mph'/>
    </current_conditions>
    <forecast_conditions>
      <day_of_week data='Sat'/>
      <low data='60'/>
      <high data='89'/>
      <icon data='/ig/images/weather/sunny.gif'/>
      <condition data='Clear'/>
    </forecast_conditions>
    <forecast_conditions>
      <day_of_week data='Sun'/>
      <low data='53'/>
      <high data='86'/>
      <icon data='/ig/images/weather/sunny.gif'/>
      <condition data='Clear'/>
    </forecast_conditions>
    <forecast_conditions>
      <day_of_week data='Mon'/>
      <low data='57'/>
      <high data='87'/>
      <icon data='/ig/images/weather/sunny.gif'/>
      <condition data='Clear'/>
    </forecast_conditions>
    <forecast_conditions>
      <day_of_week data='Tue'/>
      <low data='60'/>
      <high data='84'/>
      <icon data='/ig/images/weather/sunny.gif'/>
      <condition data='Clear'/>
    </forecast_conditions>
  </weather>
</xml_api_reply>

]]

-- we particularly want to test the built-in XML parser here, not lxp.lom
local function parse (str)
    return xml.parse(str,false,true)
end

local d = parse(joburg)


function match(t,xpect)
    local res,ret = d:match(t)
    asserteq(res,xpect,0,1) ---> note extra level, so we report on calls to this function!
end

t1 = [[
  <weather>
    <current_conditions>
      <condition data='$condition'/>
      <temp_c data='$temp'/>
    </current_conditions>
  </weather>
]]



match(t1,{
    condition = "Clear",
    temp = "24",
} )

t2 = [[
  <weather>
    {{<forecast_conditions>
      <day_of_week data='$day'/>
      <low data='$low'/>
      <high data='$high'/>
      <condition data='$condition'/>
    </forecast_conditions>}}
  </weather>
]]

local conditions = {
    {
        low = "60",
        high = "89",
        day = "Sat",
        condition = "Clear",
    },
    {
        low = "53",
        high = "86",
        day = "Sun",
        condition = "Clear",
    },
    {
        low = "57",
        high = "87",
        day = "Mon",
        condition = "Clear",
    },
    {
        low = "60",
        high = "84",
        day = "Tue",
        condition = "Clear",
    }
}

match(t2,conditions)


config = [[
<config>
    <alpha>1.3</alpha>
    <beta>10</beta>
    <name>bozo</name>
</config>
]]
d,err = parse(config)
if not d then print(err); os.exit(1) end

-- can match against wildcard tag names (end with -)
-- can be names
match([[
<config>
    {{<key->$value</key->}}
</config>
]],{
    {key="alpha", value = "1.3"},
    {key="beta", value = "10"},
    {key="name",value = "bozo"},
})
-- can be numerical indices
match([[
<config>
    {{<1->$2</1->}}
</config>
]],{
    {"alpha","1.3"},
    {"beta","10"},
    {"name","bozo"},
})
-- _ is special; means 'this value is key of captured table'
match([[
<config>
    {{<_->$1</_->}}
</config>
]],{
    alpha = {"1.3"},
    beta = {"10"},
    name = {"bozo"},
})

-- the numerical index 0 is special: a capture of {[0]=val} becomes simply the value val
match([[
<config>
    {{<_->$0</_->}}
</config>
]],{
    alpha = "1.3",
    name = "bozo",
    beta = "10"
})

-- this can of course also work with attributes, but then we don't want to collapse!

config = [[
<config>
    <alpha type='number'>1.3</alpha>
    <beta type='number'>10</beta>
    <name type='string'>bozo</name>
</config>
]]
d,err = parse(config)
if not d then print(err); os.exit(1) end

match([[
<config>
    {{<_- type='$1'>$2</_->}}
</config>
]],{
    alpha = {"number","1.3"},
    beta = {"number","10"},
    name = {"string","bozo"},
})

d,err = parse [[

<configuremap>
  <configure name="NAME" value="ImageMagick"/>
  <configure name="LIB_VERSION" value="0x651"/>
  <configure name="LIB_VERSION_NUMBER" value="6,5,1,3"/>
  <configure name="RELEASE_DATE" value="2009-05-01"/>
  <configure name="VERSION" value="6.5.1"/>
  <configure name="CC" value="vs7"/>
  <configure name="HOST" value="windows-unknown-linux-gnu"/>
  <configure name="DELEGATES" value="bzlib freetype jpeg jp2 lcms png tiff x11 xml wmf zlib"/>
  <configure name="COPYRIGHT" value="Copyright (C) 1999-2009 ImageMagick Studio LLC"/>
  <configure name="WEBSITE" value="http://www.imagemagick.org"/>

</configuremap>
]]
if not d then print(err); os.exit(1) end
--xml.debug = true

res,err = d:match [[
<configuremap>
   {{<configure name="$_" value="$0"/>}}
</configuremap>
]]

asserteq(res,{
    HOST = "windows-unknown-linux-gnu",
    COPYRIGHT = "Copyright (C) 1999-2009 ImageMagick Studio LLC",
    NAME = "ImageMagick",
    LIB_VERSION = "0x651",
    VERSION = "6.5.1",
    RELEASE_DATE = "2009-05-01",
    WEBSITE = "http://www.imagemagick.org",
    LIB_VERSION_NUMBER = "6,5,1,3",
    CC = "vs7",
    DELEGATES = "bzlib freetype jpeg jp2 lcms png tiff x11 xml wmf zlib"
})

-- short excerpt from
-- /usr/share/mobile-broadband-provider-info/serviceproviders.xml

d = parse [[
<serviceproviders format="2.0">
<country code="za">
	<provider>
		<name>Cell-c</name>
		<gsm>
			<network-id mcc="655" mnc="07"/>
			<apn value="internet">
				<username>Cellcis</username>
				<dns>196.7.0.138</dns>
				<dns>196.7.142.132</dns>
			</apn>
		</gsm>
	</provider>
	<provider>
		<name>MTN</name>
		<gsm>
			<network-id mcc="655" mnc="10"/>
			<apn value="internet">
				<dns>196.11.240.241</dns>
				<dns>209.212.97.1</dns>
			</apn>
		</gsm>
	</provider>
	<provider>
		<name>Vodacom</name>
		<gsm>
			<network-id mcc="655" mnc="01"/>
			<apn value="internet">
				<dns>196.207.40.165</dns>
				<dns>196.43.46.190</dns>
			</apn>
			<apn value="unrestricted">
				<name>Unrestricted</name>
				<dns>196.207.32.69</dns>
				<dns>196.43.45.190</dns>
			</apn>
		</gsm>
	</provider>
	<provider>
		<name>Virgin Mobile</name>
		<gsm>
			<apn value="vdata">
				<dns>196.7.0.138</dns>
				<dns>196.7.142.132</dns>
			</apn>
		</gsm>
	</provider>
</country>

</serviceproviders>
]]

res = d:match [[
    <serviceproviders>
    {{<country code="$_">
        {{<provider>
            <name>$0</name>
        </provider>}}
    </country>}}
    </serviceproviders>
]]

asserteq(res,{
  za = {
    "Cell-c",
    "MTN",
    "Vodacom",
    "Virgin Mobile"
  }
})

res = d:match [[
<serviceproviders>
 <country code="$country">
   <provider>
     <name>$name</name>
     <gsm>
      <apn value="$apn">
         <dns>196.43.46.190</dns>
      </apn>
     </gsm>
   </provider>
 </country>
</serviceproviders>
]]

asserteq(res,{
  name = "Vodacom",
  country = "za",
  apn = "internet"
})

d = parse[[
<!DOCTYPE xml>
<params>
<param>
  <name>XXX</name>
  <value></value>
</param>
<param>
  <name>YYY</name>
  <value>1</value>
</param>
</params>
]]

match([[
<params>
{{<param>
    <name>$_</name>
    <value>$0</value>
</param>}}
</params>
]],{XXX = '',YYY = '1'})


-- can always use xmlification to generate your templates...

local SP, country, provider, gsm, apn, dns = xml.tags 'serviceprovider, country, provider, gsm, apn, dns'

t = SP{country{code="$country",provider{
   name '$name', gsm{apn {value="$apn",dns '196.43.46.190'}}
   }}}

out = xml.tostring(t,' ','  ')
asserteq(out,[[

 <serviceprovider>
   <country code='$country'>
     <provider>
       <name>$name</name>
       <gsm>
         <apn value='$apn'>
           <dns>196.43.46.190</dns>
         </apn>
       </gsm>
     </provider>
   </country>
 </serviceprovider>]])

----- HTML is a degenerate form of XML ;)
-- attribute values don't need to be quoted, tags are case insensitive,
-- and some are treated as self-closing

doc = xml.parsehtml [[
<BODY a=1>
Hello dolly<br>
HTML is <b>slack</b><br>
</BODY>
]]

asserteq(xml.tostring(doc),[[
<body a='1'>
Hello dolly<br/>
HTML is <b>slack</b><br/></body>]])

doc = xml.parsehtml [[
<!DOCTYPE html>
<html lang=en>
<head><!--head man-->
</head>
<body>
</body>
</html>
]]

asserteq(xml.tostring(doc),"<html lang='en'><head/><body/></html>")

-- note that HTML mode currently barfs if there isn't whitespace around things
-- like '<' and '>' in scripts.
doc = xml.parsehtml [[
<html>
<head>
<script>function less(a,b) { return a < b; }</script>
</head>
<body>
<h2>hello dammit</h2>
</body>
</html>
]]

script = doc:get_elements_with_name 'script'
asserteq(script[1]:get_text(), 'function less(a,b) { return a < b; }')


-- test attribute order

local test_attrlist = xml.new('AttrList',{
   Attr3="Value3",
    ['Attr1'] = "Value1",
    ['Attr2'] = "Value2",
    [1] = 'Attr1', [2] = 'Attr2', [3] = 'Attr3'
})
asserteq(
xml.tostring(test_attrlist),
"<AttrList Attr1='Value1' Attr2='Value2' Attr3='Value3'/>"
)


-- commments
str = [[
<hello>
<!-- any <i>momentous</i> stuff here -->
dolly
</hello>
]]
doc = parse(str)
asserteq(xml.tostring(doc),[[
<hello>
dolly
</hello>]])


-- underscores and dashes in attributes

str = [[
<hello>
    <tag my_attribute='my_value'>dolly</tag>
</hello>
]]
doc = parse(str)

print(doc)
print(xml.tostring(doc))

asserteq(xml.tostring(doc),[[
<hello><tag my_attribute='my_value'>dolly</tag></hello>]])


-- parsing by file name

local filename = path.tmpname()
utils.writefile(filename, '<hello><world/></hello>')
doc = xml.parse(filename, true, true)
os.remove(filename)
asserteq(type(doc), 'table')
asserteq(xml.tostring(doc, '', '  ', nil, true), [[
<?xml version='1.0'?>
<hello>
  <world/>
</hello>]])


