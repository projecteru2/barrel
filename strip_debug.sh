#!/bin/bash
objcopy --only-keep-debug eru-barrel eru-barrel.debug && \
objcopy --strip-debug --add-gnu-debuglink=eru-barrel.debug eru-barrel eru-barrel.release