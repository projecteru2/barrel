#!/bin/bash
declare -a FILES=( 
  "barrel.conf" "/etc/eru/"
  "eru-barrel" "/usr/bin/"
  "eru-barrel.service" "/usr/lib/systemd/system/"
)

clear_file () {
  if [ -e "$1" ];
  then
    echo "remove $1"
    rm $1
  else
    echo "$1 not exists"
  fi  
}

copy_file () {
  echo "copy $1 to $2"
  cp $1 $2
}

if [ -d "/etc/eru" ];
then
  echo "/etc/eru exists"
else 
  echo "create /etc/eru"
  mkdir -p /etc/eru
fi

echo "===stop service==="
systemctl stop eru-barrel.service
systemctl disable eru-barrel.service

echo "===remove old files==="
for i in $(eval echo "{0..$((${#FILES[@]} - 1))..2}") 
do
  clear_file "${FILES[$((i + 1))]}${FILES[${i}]}" 
done

echo "===copy new files==="
for i in $(eval echo "{0..$((${#FILES[@]} - 1))..2}") 
do
  copy_file "${FILES[${i}]}" "${FILES[$((i + 1))]}"
done

echo "===start service==="
systemctl enable eru-barrel.service
systemctl start eru-barrel.service

echo "===barrel install success==="
