#!/usr/bin/env bash
# this script only coup with 2 situations:
#  1. tenant not release due to system crash
#  2. netns invalid due to system restart
# other uncontrolled scenarios will be labeled as bug

set -u
echo "$BARREL_CNI_BIN"
export FORCE="${FORCE:-0}"

BARREL_DIR=/var/lib/barrel/cni

for ipv4 in $(ls "$BARREL_DIR" | grep -Po '\d+(\.\d+){3}' | sort -u); do
    echo inspect "$ipv4"

    if [ -e "$BARREL_DIR/$ipv4-tenant" ]; then
        id=$(cat "$BARREL_DIR"/"$ipv4"-tenant)
        status=$(timeout 1 docker inspect "$id" -f '{{.State.Status}}' 2> /dev/null)
        [[ "$status" != "running" ]] && echo clean dead tenant "$ipv4" && rm -f "$BARREL_DIR"/"$ipv4"-tenant
    fi

    rc=$(ls -l "$BARREL_DIR" | grep "$ipv4" -c)
    if ((rc==2)); then
        owner=$(cat "$BARREL_DIR"/"$ipv4"-owner)
        echo clean dangling network "$ipv4", "$owner"
        umount "$BARREL_DIR"/"$ipv4" 2> /dev/null
        rm -f "$BARREL_DIR"/"$ipv4"-owner "$BARREL_DIR"/"$ipv4"
        "$BARREL_CNI_BIN" cni --config /etc/docker/cni.yaml --command del <<< '{"id":"'"$owner"'"}'
    fi

    mkdir -p /var/run/netns
    ln -s "$BARREL_DIR"/"$ipv4" /var/run/netns/"$ipv4"
    ip net e "$ipv4" ip a &> /dev/null
    if [[ "$?" != 0 ]]; then
        echo rebuild netns for "$ipv4"
        unshare -n python -c 'import ctypes, ctypes.util; libc = ctypes.CDLL(ctypes.util.find_library("c"), use_errno=True); libc.pause()' &
        id=$(cat "$BARREL_DIR"/"$ipv4"-owner)
        pid=$(pgrep -f '[l]ibc.pause')
        "$BARREL_CNI_BIN" cni --config /etc/docker/cni.yaml --command del <<< '{"id":"'"$id"'"}'
        CNI_ARGS=IP="$ipv4" "$BARREL_CNI_BIN" cni --config /etc/docker/cni.yaml --command add <<< '{"id":"'"$id"'","pid":'"$pid"'}'
        mount --bind /proc/"$pid"/ns/net "$BARREL_DIR"/"$ipv4"
        kill -9 "$pid"
    fi
    rm -f /var/run/netns/"$ipv4"
done
