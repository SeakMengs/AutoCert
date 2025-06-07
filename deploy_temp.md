API go service

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

Worker go service

`sudo vim /etc/systemd/system/autocertworker.service`

```
[Unit]
Description=AutoCert Worker
After=network.target

[Service]
WorkingDirectory=/var/www/AutoCert
ExecStart=/var/www/AutoCert/autocertworker
Restart=always
User=root
Group=root

[Install]
WantedBy=multi-user.target
```

```sh
sudo systemctl daemon-reload
sudo systemctl enable autocertworker
sudo systemctl start autocertworker
```

Change postgres user password

```
su postgres
psql
ALTER USER postgres WITH PASSWORD 'your_new_password';
```

Dump psql database

```
su postgres
pg_dump -U postgres -p 5432 autocert > /tmp/autocert.sql
```

Stress test clear database
```
-- Run these in a PostgreSQL session (e.g., `psql -U postgres`)

-- Step 1: Drop and recreate the database
DROP DATABASE IF EXISTS stress_test;
CREATE DATABASE stress_test;

-- Step 2: Connect to the database
\c stress_test

-- Step 3: Create the citext extension (case-insensitive text)
CREATE EXTENSION IF NOT EXISTS citext;

-- Step 4: Grant permissions and schema setup for user 'admin'

-- Grant all privileges on all existing tables
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO admin;

-- Grant usage and creation rights on the schema
GRANT USAGE, CREATE ON SCHEMA public TO admin;

-- Grant usage and SELECT on all sequences
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO admin;

-- Make 'admin' the owner of the schema
ALTER SCHEMA public OWNER TO admin;

-- Set default privileges for future tables in the schema
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO admin;
```

RabbitMQ

```sh
sudo rabbitmq-plugins enable rabbitmq_management
```

queue port: 5672
management port: 15672

Delete guest user

```sh
sudo rabbitmqctl delete_user guest
```

Create a new user

```sh
sudo rabbitmqctl add_user autocert loveeavlong
sudo rabbitmqctl set_user_tags autocert administrator
sudo rabbitmqctl set_permissions -p / autocert ".*" ".*" ".*"
```

Check user and permissions

```sh
sudo rabbitmqctl list_users
sudo rabbitmqctl list_permissions -p /
```