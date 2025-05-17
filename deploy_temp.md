`sudo vim /etc/systemd/system/autocertapi.service`

```
[Unit]
Description=AutoCert Go API
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertapi
Restart=always
# TODO: change this to the user and group that should run the service
User=root
Group=root
Environment=PORT=8080

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl daemon-reload
sudo systemctl enable autocertapi
sudo systemctl start autocertapi
```
