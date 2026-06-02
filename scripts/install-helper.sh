#!/bin/sh
set -eu

SOURCE="${1:?compiled helper path is required}"
TARGET="/usr/local/libexec/duku-capture-helper"
SUDOERS="/etc/sudoers.d/duku-net-lab"
USER_NAME="${SUDO_USER:?run through sudo}"

install -d -m 755 /usr/local/libexec
install -m 755 "$SOURCE" "$TARGET"
cat > "$SUDOERS" <<EOF
$USER_NAME ALL=(root) NOPASSWD: $TARGET probe, $TARGET start *, $TARGET start-local *, $TARGET stop, $TARGET status, $TARGET restore
EOF
chmod 440 "$SUDOERS"
visudo -cf "$SUDOERS"
echo "Installed $TARGET and validated $SUDOERS"
