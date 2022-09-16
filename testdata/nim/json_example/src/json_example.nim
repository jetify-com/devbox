# This is just an example to get you started. A typical binary package
# uses this file as the main entry point of the application.
import jsonbuilder/json_builder

when isMainModule:
  echo("running main)
  let builder = newJsonObjectBuilder()
  builder.add_entry("key", "value")

  builder.add_array("key2"):
    builder.add_entry("value")         # entries added here 
    builder.add_entry("value")         # are members of a new array

  builder.add_entry("key3", "value")   # now back in the object - so key-value pair land

  builder.add_object("key4"):
    builder.add_entry("key", "value")  # child of a new JS object
    
  builder.finish()                     # finish() adds the closing bracket
  echo builder                         # $builder returns the latest json string
