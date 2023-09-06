In this repro:
1. Running `python main.py` gives a `libstdc++` import error. Our usual recommendation
   is to do `devbox add stdenv.cc.cc.lib`. This does fix the `python main.py` error.
2. However, now running `convert` or `rsvg-convert` fail with `version GLIBC_xxx not found`.


