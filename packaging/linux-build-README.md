# Linux Debian build

This directory is a self-contained Debian source tree. On Ubuntu, install
`devscripts`, `debhelper`, `fakeroot`, `golang-go` (Go 1.24 or newer), and
`ca-certificates`, then run:

```bash
debuild -us -uc
```

The package builds the API, worker, PBS Sync, and migration binaries. It does
not automatically start services or run database migrations. Review and edit
`/etc/dhis2gw/dhis2gw.yml` after installation, then run:

```bash
sudo dhis2gw-migrate --config-file /etc/dhis2gw/dhis2gw.yml
sudo systemctl enable --now dhis2gw.service dhis2gw-worker.service
sudo systemctl enable --now dhis2gw-pbs-sync.service  # when PBS is configured
```

The Nginx site is installed at
`/etc/nginx/sites-available/nginx-dhis2gw.conf`. Enable it after reviewing
the local reverse-proxy defaults:

```bash
sudo ln -s /etc/nginx/sites-available/nginx-dhis2gw.conf /etc/nginx/sites-enabled/dhis2gw.conf
sudo nginx -t && sudo systemctl reload nginx
```
