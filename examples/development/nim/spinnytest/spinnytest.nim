import spinny, os

var spinner1 = newSpinny("Loading message ..".fgWhite, skDots)
spinner1.setSymbolColor(fgBlue)
spinner1.start

sleep(2000)

spinner1.success("Ta da! Hello World")

