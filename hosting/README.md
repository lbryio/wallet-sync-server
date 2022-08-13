These are some simple instructions to get your lbry wallet sync server running. This is mostly a note to self and suggestion to others, but you can host however you want. We use supervisord and the Caddy web server. Ideally we'll probably switch to systemd and we'd include a Caddyfile.

Install the latest version of golang. Check out this repository, and run `go build .` in the repository root.

Insteall the [caddy web server](https://caddyserver.com/). The website has its own debian repos. You can also install from source (golang). Which ever (if either) you feel comfortable with. No specific recommendations here.

You'll need to adjust your firewall to allow http (port 80) for caddy to obtain an SSL cert via ACME (ZeroSSL or LetsEncrypt) and also allow https (port 443) to have caddy serve our wallet sync server. If you use `ufw`:

```
sudo ufw allow http
sudo ufw allow https
```

To avoid running Caddy as root, you'll need to allow it to serve from ports 80 and 443 ([see here](https://superuser.com/a/892391)):

```
sudo setcap 'cap_net_bind_service=+ep' /home/lbry/caddy/cmd/caddy/caddy
```

Finally, included are a couple quick conf scripts for supervisord to help get lbry.id running. Fill in the `{blank}`s and put under `/etc/supervisor/conf.d/`:

* `{caddy-cmd}`: The command to run caddy. If you're building from source, it will be `{caddy git repo root}/cmd/caddy/caddy`. If you're installing from a debian repo it's probably just `caddy`, but refer to the Caddy docs.
* `{lbry-user}`: The user you're using to run Caddy and the sync server.
* `{lbry-user-home}`: The home directory of `{lbry-user}`.
* `{sync-server-dir}`: The git repo root of the lbry sync server.
