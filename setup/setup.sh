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

read -rp "Please enter the database name for archon (default: archondb): " DB_NAME
if [ ! "$DB_NAME" ]; then
  DB_NAME="archondb"
fi

read -rp "Please enter the username for the archon database (default: archonadmin): " ARCHON_USER
if [ ! "$ARCHON_USER" ]; then
  ARCHON_USER="archonadmin"
fi

read -rp "Please enter the password for the archon database (default: psoadminpassword): " ARCHON_PASSWORD
if [ ! "$ARCHON_PASSWORD" ]; then
  ARCHON_PASSWORD="psoadminpassword"
fi

read -rp "Please enter the server address (default: 0.0.0.0): " SERVER_IP
if [ ! "$SERVER_IP" ]; then
  SERVER_IP="0.0.0.0"
fi

read -rp "Please enter the external server address (default: 127.0.0.1): " EXTERNAL_ADDRESS
if [ ! "$EXTERNAL_ADDRESS" ]; then
  EXTERNAL_ADDRESS="127.0.0.1"
fi

SETUP_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )"

# The user can provide the install location as the first option
# If it's not provided, we'll use the root archon dir.
if [ -z "$1" ]; then
  pushd "$SETUP_DIR" > /dev/null 2>&1
  cd ..
  mkdir archon_server
  cd archon_server
  INSTALL_DIR=$(pwd)
else
  INSTALL_DIR="$1"
fi

if [ ! -d "$INSTALL_DIR" ]; then
  mkdir "$INSTALL_DIR" || echo "Please enter a valid directory."
fi

pushd "$INSTALL_DIR" > /dev/null 2>&1

mkdir .bin
export GOBIN="$(pwd)/.bin"

pushd "$SETUP_DIR"/.. > /dev/null 2>&1
go install ./cmd/*
popd > /dev/null 2>&1

# Copy compiled binaries to the server folder.
cp .bin/* "$INSTALL_DIR"

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

# Edit hostname
SEARCH='hostname: 0.0.0.0'
REPLACE="hostname: $SERVER_IP"
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

# Edit external address
SEARCH='external_ip: 127.0.0.1'
REPLACE="external_ip: $EXTERNAL_ADDRESS"
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

# Edit certificate location
SEARCH='shipgate_certificate_file: "certificate.pem"'
REPLACE="shipgate_certificate_file: \"$(pwd)/certificate.pem\""
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

# Edit key location
SEARCH='ssl_key_file: "key.pem"'
REPLACE="ssl_key_file: \"$(pwd)/key.pem\""
sed -i '' "s#$SEARCH#$REPLACE#" config.yaml

createdb "$DB_NAME"
psql $DB_NAME -c "CREATE USER $ARCHON_USER WITH ENCRYPTED PASSWORD '$ARCHON_PASSWORD';"
psql $DB_NAME -c "GRANT ALL ON ALL TABLES IN SCHEMA public TO $ARCHON_USER;"

# This should exist, but let's verify just in case.
if [ ! -d patches ]; then
  mkdir patches
  cp -r patches/* patches/.
fi


echo "Generating certificates..."
./generate_cert --ip "$SERVER_IP" > /dev/null 2>&1
echo "Done."

echo "Adding account..."
./account --config . add
echo "Done."

echo
echo "Archon setup is complete."
echo
echo "If there are patch files you would like the server to verify, please copy them into:"
echo "  $(pwd)/patches"
echo
echo "Please verify the config file has the correct settings before running."
echo "To run the server, execute the following:"
echo "  $(pwd)/server --config $(pwd)"
echo
