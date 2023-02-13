<?php

// Check that the extension is loaded.
if (!extension_loaded('ds')) {
    echo("ds extension is not enabled\n");
    exit(1);
}

$vec = new \Ds\Vector(["hello", "world"]);
  
echo("Original vector elements\n");
foreach ($vec as $idx => $elem) {
  echo("idx: $idx and elem: $elem\n");
}
echo("done\n");
