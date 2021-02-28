#!/bin/bash
set -e

if ! command -v go >/dev/null 2>&1
then
    echo "Please install Go."
    echo "Instructions can be found here: https://golang.org/"
    exit 1
fi

if ! command -v git >/dev/null 2>&1
then
    echo "Please install Git."
    echo "Instructions can be found here: "
    exit 1
fi

if ! command -v psql >/dev/null 2>&1
then
    echo "Please install Postgresql."
    echo "Instructions can be found here: https://www.postgresql.org/"
    exit 1
fi

read -rp "Please enter the database name for archon (default: archondb)" DB_NAME
if [ ! "$DB_NAME" ]; then
  DB_NAME="archondb"
fi

read -rp "Please enter the username for the archon database (default: archonadmin)" ARCHON_USER
if [ ! "$ARCHON_USER" ]; then
  ARCHON_USER="archonadmin"
fi

read -rp "Please enter the password for the archon database (default: psoadminpassword)" ARCHON_PASSWORD
if [ ! "$ARCHON_PASSWORD" ]; then
  ARCHON_PASSWORD="psoadminpassword"
fi

read -rp "Please enter the server address and port (default: 0.0.0.0/32)" SERVER_IP
if [ ! "$SERVER_IP" ]; then
  SERVER_IP="0.0.0.0/32"
fi

# No matter where we're calling this from, we're going to use
# The root archon directory as our install location.
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# Go to the root archon directory.
pushd "$SCRIPT_DIR"/.. > /dev/null 2>&1

SETUP_DIR="$(pwd)/setup"

echo "Please edit the config file: $SETUP_DIR/config.yaml"
echo "This script will edit the parameters_dir and patch_dir appropriately."
read -rp "Press enter when you're done to continue."
echo "Continuing..."

mkdir .bin
export GOBIN="$(pwd)/.bin"
go install ./cmd/*

mkdir archon_server
# Copy compiled binaries to the server folder.
cp .bin/* archon_server/.

pushd archon_server > /dev/null 2>&1
# Copy all setup files to the server folder.
cp -r "$SETUP_DIR"/* .

# Edit default patches directory.
SEARCH='patch_dir: "/usr/local/etc/archon/patches"'
REPLACE="patch_dir: \"$(pwd)/patches\""
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

# Edit default parameters directory
SEARCH='parameters_dir: "/usr/local/etc/archon/parameters"'
REPLACE="parameters_dir: \"$(pwd)/parameters\""
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

createdb "$DB_NAME"
psql archondb -c "CREATE USER $ARCHON_USER WITH ENCRYPTED PASSWORD '$ARCHON_PASSWORD';"
psql archondb -c "GRANT ALL ON ALL TABLES IN SCHEMA public TO $ARCHON_USER;"

# This should exist, but let's verify just in case.
if [ ! -d patches ]; then
  mkdir patches
  cp -r patches/* patches/.
fi

echo "If there are patch files you would like the server to verify, please copy them into:"
echo "$(pwd)/patches"
read -rp "Press enter when you're done to continue."
echo "Continuing..."

echo "Generating certificates..."
./generate_cert --ip "$SERVER_IP" > /dev/null 2>&1
echo "Done."

echo "Adding account..."
./account --config . add
echo "Done."

echo "Starting server..."
./server --config .
