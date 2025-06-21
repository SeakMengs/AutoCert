# Production Deployment Guide

This guide provides step-by-step instructions for deploying the application in a production environment. using Ubuntu 22.04.

# Pull the code

Assuming we pull the code to `/var/www/AutoCert`, you can use the following commands:

```bash
git clone --recursive https://github.com/SeakMengs/AutoCert.git /var/www/AutoCert
cd /var/www/AutoCert
```

# Install go

```
sudo apt update
sudo apt install golang -y
```

# Database Setup

sudo apt update
sudo apt install postgresql postgresql-contrib -y

sudo systemctl start postgresql
sudo systemctl enable postgresql

sudo su - postgres
psql
CREATE DATABASE autocert;

<!-- use that database first -->

\c autocert

<!-- Then -->

CREATE EXTENSION IF NOT EXISTS citext;

CREATE USER admin WITH LOGIN PASSWORD 'adminadmin';

-- Grant all privileges on tables in the public schema:
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO admin;

-- Grant usage and creation rights on the public schema:
GRANT USAGE, CREATE ON SCHEMA public TO admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public to admin;

-- Optionally, assign ownership of the public schema to the user:
ALTER SCHEMA public OWNER TO admin;

-- Grant default privileges for future objects in the schema:
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO admin;

vim /etc/postgresql/16/main/postgresql.conf
listen_addresses = '\*'

vim /etc/postgresql/16/main/pg_hba.conf
IPv4
0.0.0.0/0

peer -> md5

## Migrate the database

first update .env file

```bash
cp /var/www/AutoCert/.env.example /var/www/AutoCert/.env
```

fill in the database connection details in `.env` file:

then migrate using the following command:

```bash
go run /var/www/AutoCert/cmd/migrate/main.go
```

# Set up minio for our self-hosted S3

For latest version go to this [minio linux release page](https://min.io/docs/minio/linux/index.html)

```bash
wget https://dl.min.io/server/minio/release/linux-amd64/archive/minio_20250422221226.0.0_amd64.deb -O minio.deb
sudo dpkg -i minio.deb
```

## Create the Environment Variable File

Create user and group for minio:

```bash
groupadd -r minio-user
useradd -M -r -g minio-user minio-user
chown minio-user:minio-user /mnt/data
```

Create a file named `/etc/default/minio` and add the following content:

```bash
# MINIO_ROOT_USER and MINIO_ROOT_PASSWORD sets the root account for the MinIO server.
# This user has unrestricted permissions to perform S3 and administrative API operations on any resource in the deployment.
# Omit to use the default values 'minioadmin:minioadmin'.
# MinIO recommends setting non-default values as a best practice, regardless of environment

MINIO_ROOT_USER=myminioadmin
MINIO_ROOT_PASSWORD=minio-secret-key-change-me

# MINIO_VOLUMES sets the storage volume or path to use for the MinIO server.

MINIO_VOLUMES="/mnt/data"

# MINIO_OPTS sets any additional commandline options to pass to the MinIO server.
# For example, `--console-address :9001` sets the MinIO Console listen port
MINIO_OPTS="--console-address :9001"
```

Enable and start the MinIO service:

```bash
sudo systemctl enable minio
sudo systemctl start minio
```

confirm MinIO is running by checking the status:

```bash
sudo systemctl status minio.service
journalctl -f -u minio.service
```

<!-- TODO: generate secret key and access key -->

# Configure rabbitmq

For latest version go to this [rabbitmq linux release page](https://www.rabbitmq.com/docs/install-debian)

```bash
#!/bin/sh

sudo apt-get install curl gnupg apt-transport-https -y

## Team RabbitMQ's main signing key
curl -1sLf "https://keys.openpgp.org/vks/v1/by-fingerprint/0A9AF2115F4687BD29803A206B73A36E6026DFCA" | sudo gpg --dearmor | sudo tee /usr/share/keyrings/com.rabbitmq.team.gpg > /dev/null
## Community mirror of Cloudsmith: modern Erlang repository
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-erlang.E495BB49CC4BBE5B.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg > /dev/null
## Community mirror of Cloudsmith: RabbitMQ repository
curl -1sLf https://github.com/rabbitmq/signing-keys/releases/download/3.0/cloudsmith.rabbitmq-server.9F4587F226208342.key | sudo gpg --dearmor | sudo tee /usr/share/keyrings/rabbitmq.9F4587F226208342.gpg > /dev/null

## Add apt repositories maintained by Team RabbitMQ
sudo tee /etc/apt/sources.list.d/rabbitmq.list <<EOF
## Provides modern Erlang/OTP releases
##
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

# another mirror for redundancy
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.E495BB49CC4BBE5B.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-erlang/deb/ubuntu noble main

## Provides RabbitMQ
##
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa1.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main

# another mirror for redundancy
deb [arch=amd64 signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
deb-src [signed-by=/usr/share/keyrings/rabbitmq.9F4587F226208342.gpg] https://ppa2.rabbitmq.com/rabbitmq/rabbitmq-server/deb/ubuntu noble main
EOF

## Update package indices
sudo apt-get update -y

## Install Erlang packages
sudo apt-get install -y erlang-base \
                        erlang-asn1 erlang-crypto erlang-eldap erlang-ftp erlang-inets \
                        erlang-mnesia erlang-os-mon erlang-parsetools erlang-public-key \
                        erlang-runtime-tools erlang-snmp erlang-ssl \
                        erlang-syntax-tools erlang-tftp erlang-tools erlang-xmerl

## Install rabbitmq-server and its dependencies
sudo apt-get install rabbitmq-server -y --fix-missing
```

Delete guest user

```sh
sudo rabbitmqctl delete_user guest
```

Create a new user

```sh
sudo rabbitmqctl add_user admin adminpw
sudo rabbitmqctl set_user_tags admin administrator
sudo rabbitmqctl set_permissions -p / admin ".*" ".*" ".*"
```

Check user and permissions

```sh
sudo rabbitmqctl list_users
sudo rabbitmqctl list_permissions -p /
```

Enable queue management plugin

```sh
sudo rabbitmq-plugins enable rabbitmq_management
```

# Configure the application

## Download node js and npm

Install node js and npm

For latest version go to this [nodejs release page](https://nodejs.org/en/download)

```bash
# Download and install nvm:
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.40.3/install.sh | bash

# in lieu of restarting the shell
\. "$HOME/.nvm/nvm.sh"

# Download and install Node.js:
nvm install 22

# Verify the Node.js version:
node -v # Should print "v22.16.0".
nvm current # Should print "v22.16.0".

# Verify npm version:
npm -v # Should print "10.9.2".
```

Install pm2

```bash
npm install -g pm2
```

## Configure API

Set up AutoCert API service. Besure to fill in the `.env` file with the correct values.

Note: for minio, ACCESS_KEY and SECRET_KEY can be the minio root user and password you set in `/etc/default/minio`.

```bash

build the API binary:

``` bash
go build -o /var/www/AutoCert/autocertapi ./cmd/api/main.go

````

```bash
sudo vim /etc/systemd/system/autocertapi.service
````

Add the following content to the file:

```ini
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

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertapi
sudo systemctl start autocertapi
```

## Configure Certificate Worker

```bash
go build -o /var/www/AutoCert/autocertcertworker ./cmd/cert_consumer/main.go
```

```bash
sudo vim /etc/systemd/system/autocertcertworker.service
```

Add the following content to the file:

```ini
[Unit]
Description=AutoCert Certificate Worker
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertcertworker
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertcertworker
sudo systemctl start autocertcertworker
```

## Configure Mail Worker

```bash
go build -o /var/www/AutoCert/autocertmailworker ./cmd/mail_consumer/main.go
```

```bash
sudo vim /etc/systemd/system/autocertmailworker.service
```

Add the following content to the file:

```ini
[Unit]
Description=AutoCert Mail Worker
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertmailworker
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable autocertmailworker
sudo systemctl start autocertmailworker
```

## Configure Frontend

Fill in the `.env` file with the correct values.

```bash
cd /var/www/AutoCert/web
cp .env.example .env
```

Install dependencies and build the frontend:

```bash
npm install --force
npm run build
```

Use pm2 to run the frontend:

```bash
pm2 start npm --name "autocert-frontend" -- run start
pm2 save
pm2 startup
```

### Configure Nginx

The following is an example Nginx configuration file for serving the AutoCert frontend and proxying API requests:

```
# Catch-all server block to reject unknown domains
server {
    listen 80 default_server;
    server_name _;

    return 444; # Nginx's special "no response" code (can also use 403)
}

# =======================
# MinIO Console
# =======================
server {
    listen 80;
    server_name autocert-console.scormetry.site;

    location / {
        proxy_pass http://localhost:9001;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_cache_bypass $http_upgrade;
    }
}

# =======================
# MinIO S3 API
# =======================
server {
   listen       80;
   listen  [::]:80;
   server_name autocert-storage.scormetry.site;

   # Allow special characters in headers
   ignore_invalid_headers off;
   # Allow any size file to be uploaded.
   # Set to a value such as 1000m; to restrict file size to a specific value
   client_max_body_size 0;
   # Disable buffering
   proxy_buffering off;
   proxy_request_buffering off;

   location / {
      proxy_set_header Host $http_host;
      proxy_set_header X-Real-IP $remote_addr;
      proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
      proxy_set_header X-Forwarded-Proto $scheme;

      proxy_connect_timeout 300;
      # Default is HTTP/1, keepalive is only enabled in HTTP/1.1
      proxy_http_version 1.1;
      proxy_set_header Connection "";
      chunked_transfer_encoding off;

      proxy_pass http://localhost:9000;
   }
}

# =======================
# Backend API
# =======================
server {
    listen 80;
    server_name autocert-api.scormetry.site;
    client_max_body_size 50m;

    location / {
        proxy_pass http://localhost:8080;
        # For getting client ip
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        # Pass the original protocol scheme (HTTP/HTTPS) to the backend
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# =======================
# Frontend App
# =======================
server {
    listen 80;
    server_name autocert.scormetry.site;
    client_max_body_size 50m;

    location / {
        proxy_pass http://localhost:3000;
        # For getting client ip
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        # Pass the original protocol scheme (HTTP/HTTPS) to the backend
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}

# =======================
# RabbitMq
# =======================
server {
    listen 80;
    server_name autocert-queue.scormetry.site;

    location / {
        proxy_pass http://localhost:15672;
        # For getting client ip
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
        # Pass the original protocol scheme (HTTP/HTTPS) to the backend
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

# TIP

To see live logs of the services, you can use the following commands:

```bash
sudo journalctl -f -u autocertapi.service
sudo journalctl -f -u autocertcertworker.service
sudo journalctl -f -u autocertmailworker.service
```