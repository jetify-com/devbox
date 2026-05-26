<?php

if (!extension_loaded('ds')) {
    fwrite(STDERR, "ds extension is not enabled\n");
    exit(1);
}

$seq = new \Ds\Seq(["hello", "world"]);

echo "Original sequence elements\n";
foreach ($seq as $idx => $elem) {
    echo "idx: $idx and elem: $elem\n";
}
echo "done\n";
