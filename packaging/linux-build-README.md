# Linux Debian build

This directory is a self-contained Debian packaging tree. The supported build
host is Ubuntu 24.04 LTS (Noble). Install the packaging tools and the Go
toolchain used by the repository before building:

```bash
sudo apt update
sudo apt install devscripts debhelper fakeroot ca-certificates
```

The repository requires Go 1.24.3 or newer. GoReleaser 2.5.1 or newer is also
required. Verify both tools before building:

```bash
go version
goreleaser --version
```

From the repository root, generate the Linux/amd64 binaries and Debian source
tree with:

```bash
make linux-build
```

The target creates one self-contained Linux/amd64 tree:

```bash
cd linux-build/amd64
debuild -us -uc
```

The package contains the GoReleaser-built API, worker, PBS Sync, and migration
binaries. Debian debhelper installs the service units using Ubuntu's merged-
`/usr` systemd layout; do not add a hard-coded systemd executable path to the
package. The package does not automatically start services or run database
migrations.
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
