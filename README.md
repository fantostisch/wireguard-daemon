# WireGuard Daemon

Daemon for managing a WireGuard server using an API.
Built for [eduVPN](https://eduvpn.org).

This project is used by the
[eduVPN portal with WireGuard support](https://github.com/fantostisch/vpn-user-portal).

## API endpoints overview

| Method | URL                         | POST Data                              | Description                                                                                                  |
|--------|-----------------------------|----------------------------------------|--------------------------------------------------------------------------------------------------------------|
| GET    | /configs?user_id=foo        |                                        | List all configs of the user. Return empty list if no configs found.                                         |
| POST   | /create_config              | user_id=foo&public_key=ABC             | Create client config. Creating 2 client configs with the same public key will overwrite the existing config. |
| POST   | /create_config_and_key_pair | user_id=foo                            | Create client config. Let the server create a public private key pair.                                       |
| POST   | /delete_config              | user_id=foo&public_key=ABC             | Delete client config. Responds config_not_found  error if config not found.                                  |
| GET    | /client_connections         |                                        | Get clients that successfully send or received a packet in the last 3 minutes.                               |
| POST   | /disable_user               | user_id=foo                            | Disable user. Responds user_already_disabled error if user is already disabled.                              |
| POST   | /enable_user                | user_id=foo                            | Enable user. Responds user_already_enabled error if user is already enabled.                                 |

todo: document return values including errors

## Compatibility

### Debian 10 (Buster)
WireGuard, Go and systemd must be installed from backports, which needs to be enabled. [Instructions for enabling backports on Debian](https://backports.debian.org/Instructions/).
```sh
sudo apt install -t buster-backports wireguard golang-1.14-go systemd
```

### Completely working
* Debian 11 (Bullseye)
* Debian Unstable (Sid)

## Development

### Installation

If the linux kernel headers are not already installed, a reboot might be necessary.
```sh
sudo apt update
sudo apt -y install wireguard linux-headers-generic golang-go

git clone https://github.com/fantostisch/wireguard-daemon.git
cd wireguard-daemon
(cd deploy && bash ./deploy.sh 51820)
sudo setcap cap_net_admin=ep _bin/wireguard-daemon
_bin/wireguard-daemon --init --storage-file _bin/storage.json
```

#### Running
```sh
make run
```

### Set up NAT

Execute the following and replace `eth0` with your primary network interface which you can find by executing `sudo ifconfig`.
```sh
sudo iptables -t nat -A POSTROUTING -s 10.0.0.0/8 -o eth0 -j MASQUERADE
```

### Uninstall

```
# Warning: removes all data, including all configurations.
(cd deploy && bash ./purge.sh)
```

## Deploying eduVPN with WireGuard support
**Warning**: until version 1.x is released no data migrations will be provided and all data will be lost on update. \
**Warning**: the eduVPN repository will be replaced with a new repository which is not officially supported by eduVPN.

First [deploy eduVPN on Debian Bullseye](https://github.com/eduvpn/documentation/blob/v2/DEPLOY_DEBIAN.md).
Then run the following:
```sh
# Replace firewall rules
curl https://raw.githubusercontent.com/fantostisch/documentation/v2/resources/firewall/iptables | sudo tee /etc/iptables/rules.v4 > /dev/null

# Add software repository
gpg --keyserver pgp.surfnet.nl --recv-keys D5AF93E78144D01912ED0BA6A54D467852A97D0E
gpg --export D5AF93E78144D01912ED0BA6A54D467852A97D0E | sudo tee /etc/apt/trusted.gpg.d/eduVPN-WireGuard.gpg > /dev/null
sudo rm /etc/apt/sources.list.d/eduVPN.list
#todo: https
echo "deb http://eduvpn-wireguard-debian.nickaquina.nl/repo sid main" | sudo tee /etc/apt/sources.list.d/eduVPN-WireGuard.list

# Replace packages
sudo apt update
sudo apt -y install wireguard-daemon wireguard-vpn-user-portal wireguard-vpn-server-api wireguard-vpn-server-node

sudo deploy-wireguard-daemon

sudo systemctl enable --now wireguard-daemon
sudo systemctl start wireguard-daemon

sudo a2enconf vpn-server-api vpn-user-portal
sudo systemctl reload apache2
```
Add the following to `/etc/vpn-user-portal/config.php` and replace `vpn.example` with the hostname of your server:
```
'WireGuard' => [
    'enabled' => true,
    'hostName' => 'vpn.example',
    'dns' => ['9.9.9.9'],
  ],
```
For more options and an explanation take a look at [an example config](https://github.com/fantostisch/wireguard-vpn-user-portal/blob/c96685219a0f29066948dacd80a49db5b7a82e0f/config/config.php.example#L185).

View the daemon log with:
```sh
sudo journalctl -f -t wireguard-daemon
```

## Manage WireGuard

### Disable WireGuard
```sh
sudo ip link set down wg0
```

### Enable WireGuard
```sh
sudo ip link set up wg0
```

## License

**License**:  AGPL-3.0-or-later

```
This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program.  If not, see <https://www.gnu.org/licenses/>.
```
