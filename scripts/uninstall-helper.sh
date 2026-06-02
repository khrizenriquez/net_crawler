#!/bin/sh
set -eu

rm -f /etc/sudoers.d/duku-net-lab
rm -f /usr/local/libexec/duku-capture-helper
echo "Removed Duku Net Lab capture helper"

