#!/bin/sh
# Guest opener for dc-files. Write abs paths to the host-watched queue and exit.
# Must return immediately so yazi block=true does not freeze.
# Queue path comes from the host (world-writable) so remoteUser can append.
q=${DC_OPEN_QUEUE:-/tmp/dc-cli-open.q}
for f in "$@"; do
  case $f in
    /*) p=$f ;;
    *) p=`pwd`/$f ;;
  esac
  case $p in
    *'
'*) continue ;;
  esac
  printf '%s\0' "$p" >>"$q"
done
exit 0
