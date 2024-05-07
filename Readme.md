# simple-dyndns
Simple web application to accept dynamic updates by f.e. a fritzbox and set an a name record via cloudflares dns api.

## Install

1. go build
1. mkdir /opt/simple-dyndns
1. put binary inside /opt/simple-dyndns
1. put simple-dyndns.service inside /etc/systemd/system
1. systemctl daemon-reload
1. systemctl start simple-dyndns.service (will fail because no config, but DynamicUser folders are created)
1. create config under /var/lib/private/simple-dyndns/config.json
1. (optional) symlink /etc/simple-dyndns/config.json to /var/lib/private/simple-dyndns/config.json
1. systemctl enable --now simple-dyndns.service
1. create a reverse proxy to expose http://127.0.1.1:8080 to the internet

## Configure FritzBox

| Field        | Value                                                                  |
|--------------|------------------------------------------------------------------------|
| Update-URL   | `https://%hostname%(/%path%)?fqdn=<domain>&token=<pass>&ipv4=<ipaddr>` |
| Domainname   | The hostname that should be updated                                    |
| Benutzername | not used, just put `none`                                              |
| Kennwort     | The web_token for the hostname                                         |
