# SimpleFRP Backup and Recovery

## Source Code

The GitHub repository is the source-code backup:

```bash
git clone https://github.com/QiaoxiuLi/SimpleFRP.git
cd SimpleFRP
make test
make release
```

Use this normal update flow:

```bash
git pull --rebase
git add .
git commit -m "Describe the change"
git push
```

## Release Artifacts

Release packages are attached to GitHub Releases. Download the latest release from:

```text
https://github.com/QiaoxiuLi/SimpleFRP/releases
```

## Runtime Configuration

Runtime files are not committed to git because they contain server/client secrets and local state.

Back up a deployed Linux server or client:

```bash
sudo tar -czf simplefrp-runtime-backup.tar.gz /etc/simplefrp /var/lib/simplefrp
```

Restore:

```bash
sudo tar -xzf simplefrp-runtime-backup.tar.gz -C /
sudo systemctl restart simplefrp-server || sudo systemctl restart simplefrp-client
```
