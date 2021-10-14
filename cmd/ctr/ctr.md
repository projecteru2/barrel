## barrel-utils inspect block
show info about the block
allocated ips
unallocated ips

## barrel-utils inspect ip
show info about the ip
fixed or not
assigned or not

## barrel-utils diag ip 
show info about the ip
fixed or not
assigned or not
leaked or not
related leaked wep
parameters: --release --force

## barrel-utils diag pool
show info about the pool
blocks in pool(count)
empty blocks in pool --show-empty-blocks
weps of pool(count)
leaked weps of pool --show-leaed-weps
assigned ips of pool(count)
leaked ips of pool

## barrel-utils diag host
show infos about the host
default specific localhost
blocks used
empty blocks in pool --show-empty-blocks
weps of pool(count)
leaked weps of pool --show-leaed-weps
assigned ips of pool(count)
leaked ips of pool

flag: --pool consider not needed later

## barrel-utils release ip
assigned fixed-ip -> unassigned fixed-ip
unassigned fixed-ip -> return to calico
assigned ip -> return to calico

arg0: ipv4
flag: pool must provided
flag: --fixed-ip-only only operate on fixed ip
flag: --clear-fixed-ip assgned fixed-ip will return to calico

## barrel-utils release block
release block affinity and delete block

arg0: block cidr
flag: --pool 

## barrel-utils release blocks
release block affinity and delete blocks

arg0: pool

## barrel-utils release wep
arg0: wep name
flag: --pool