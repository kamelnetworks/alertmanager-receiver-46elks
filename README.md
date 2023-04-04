# alertmanager-receiver-46elks
SMS sender using the 46elks API for Prometheus Alertmanager

* compile program using 1.19 or better: go build main.go
* copy program to /usr/local/bin/
* copy env file to /etc/sysconfig
* alter env file to suide your needs
* copy servicefile to /etc/systemd/system/
* reload system do to read service file: systemctl daemon-reload
* enable service: systemctl enable alertmanager-receiver-46elks.service
* start service: systemctl start alertmanager-receiver-46elks.service

