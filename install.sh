cp --parents /eru/barrel.conf /etc/
cp eru-barrel.service /usr/lib/systemd/system/
systemctl enable eru-barrel.service
systemctl start eru-barrel.service

