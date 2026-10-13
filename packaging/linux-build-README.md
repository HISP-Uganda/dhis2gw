# Linux Debian build

This directory is a self-contained Debian packaging tree. On Ubuntu, install
`devscripts`, `debhelper`, `fakeroot`, and `ca-certificates`. From the
repository root, first generate the Linux/amd64 and Linux/arm64 binaries and
Debian source trees with:

The release targets require GoReleaser 2.5.1 or newer. Verify the installed
version with `goreleaser --version` before building.

```bash
make linux-build
```

The target creates two self-contained trees. Use the tree matching the Linux
host architecture:

```bash
cd linux-build/amd64  # use linux-build/arm64 on an arm64 host
debuild -us -uc
```

The package contains the GoReleaser-built API, worker, PBS Sync, and migration
binaries. It does not automatically start services or run database migrations.
Review and edit
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
