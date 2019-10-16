set -e

echo "run this as root"

ls package.json

chown root node_modules/electromon/node_modules/electron/dist/chrome-sandbox
chown root node_modules/electron/dist/chrome-sandbox
chmod 4755 node_modules/electromon/node_modules/electron/dist/chrome-sandbox
chmod 4755 node_modules/electron/dist/chrome-sandbox
