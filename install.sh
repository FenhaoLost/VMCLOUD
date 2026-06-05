#!/bin/bash
set -euo pipefail

echo "====================================="
echo "  CLICD Installation"
echo "====================================="

if [ "$EUID" -ne 0 ]; then
    echo "Please run as root: sudo ./install.sh"
    exit 1
fi

if ! command -v lxc-create >/dev/null 2>&1; then
    echo "LXC is not installed. Installing dependencies..."
    if command -v apt-get >/dev/null 2>&1; then
        apt-get update
        apt-get install -y lxc lxc-templates bridge-utils xz-utils quota
    elif command -v yum >/dev/null 2>&1; then
        yum install -y epel-release
        yum install -y lxc lxc-templates xz quota
    elif command -v dnf >/dev/null 2>&1; then
        dnf install -y lxc lxc-templates xz quota
    else
        echo "Could not detect package manager. Please install LXC manually."
        exit 1
    fi
fi

# Setup subordinate UID/GID for unprivileged containers
echo "Setting up subordinate UID/GID ranges..."
grep -q '^root:' /etc/subuid 2>/dev/null || echo 'root:100000:65536' >> /etc/subuid
grep -q '^root:' /etc/subgid 2>/dev/null || echo 'root:100000:65536' >> /etc/subgid

# Enable ext4 project quota if supported
if tune2fs -l /dev/sda1 2>/dev/null | grep -q 'Filesystem features'; then
    echo "Enabling ext4 project quota..."
    mkdir -p /etc/initramfs-tools/hooks /etc/initramfs-tools/scripts/local-premount
    
    # Hook to copy tune2fs into initramfs
    cat > /etc/initramfs-tools/hooks/tune2fs-hook << 'HOOK'
#!/bin/sh
PREREQ=""
prereqs() { echo "$PREREQ"; }
case "$1" in prereqs) prereqs; exit 0;; esac
. /usr/share/initramfs-tools/hook-functions
copy_exec /sbin/tune2fs /sbin/tune2fs
copy_exec /usr/sbin/setquota /usr/sbin/setquota
HOOK
    chmod +x /etc/initramfs-tools/hooks/tune2fs-hook
    
    # Script to run tune2fs before mount
    cat > /etc/initramfs-tools/scripts/local-premount/prjquota << 'SCRIPT'
#!/bin/sh
PREREQ=""
prereqs() { echo "$PREREQ"; }
case "$1" in prereqs) prereqs; exit 0;; esac
/sbin/tune2fs -O project -Q prjquota /dev/sda1 2>/dev/null
SCRIPT
    chmod +x /etc/initramfs-tools/scripts/local-premount/prjquota
    
    update-initramfs -u -k all 2>/dev/null || true
    
    # Add prjquota to fstab if not already there
    grep -q 'prjquota' /etc/fstab 2>/dev/null || sed -i 's|ext4 rw,|ext4 rw,prjquota,|' /etc/fstab
fi

if [ ! -f "./clicd" ]; then
    echo "ERROR: clicd binary not found in current directory"
    exit 1
fi

cp ./clicd /usr/local/bin/clicd
chmod +x /usr/local/bin/clicd
echo "Installed binary: /usr/local/bin/clicd"

cat > /etc/systemd/system/clicd.service << 'EOF'
[Unit]
Description=CLICD - LXC Container Manager
After=network.target lxc.service

[Service]
Type=simple
ExecStart=/usr/local/bin/clicd server
Restart=always
RestartSec=5
Environment=PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable clicd
systemctl restart clicd

sleep 2

echo ""
echo "====================================="
echo "  Installation Complete"
echo "====================================="
echo "  Web: http://YOUR_SERVER_IP:8999"
echo "  Service: systemctl {start|stop|restart|status} clicd"
echo "  Logs: journalctl -u clicd -f"
echo "====================================="
echo ""
echo "Initial credentials, if this was the first run:"
journalctl -u clicd --no-pager -n 80 | grep -E "Username:|Password:" || true
echo ""
echo "If no password is shown, this server already had /root/.clicd/config.json."
echo "The existing admin password cannot be recovered from the bcrypt hash."
