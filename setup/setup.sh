#!/bin/bash
set -e

function sed_replace
{
  SEARCH="$1"
  REPLACE="$2"
  FILE="$3"

  # Determine which linux platform we are running on.
  # This is needed due to differences in the sed implementation in bsd and gnu
  unamestr=$(uname)
  if [ "$unamestr" = 'Linux' ]; then
    sed -i "s#$SEARCH#$REPLACE#" "$FILE"
  elif [ "$unamestr" = 'FreeBSD' ]; then
    sed -i '' "s#$SEARCH#$REPLACE#" "$FILE"
  elif [ "$unamestr" = 'Darwin' ]; then
    sed -i '' "s#$SEARCH#$REPLACE#" "$FILE"
  else
    echo "Unknown Platform...exiting."
    exit 1
  fi
}

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

read -rp "Is this the first time setup? (default: y): " FIRST_TIME_SETUP
if [ ! "$FIRST_TIME_SETUP" ]; then
  FIRST_TIME_SETUP="y"
fi

read -rp "Is this setup for docker? (default: n): " DOCKER
if [ ! "$DOCKER" ]; then
  DOCKER="n"
fi

read -rp "Please enter the server address (default: 127.0.0.1): " SERVER_IP
if [ ! "$SERVER_IP" ]; then
  SERVER_IP="127.0.0.1"
fi

read -rp "Please enter the external server address (default: 127.0.0.1): " EXTERNAL_ADDRESS
if [ ! "$EXTERNAL_ADDRESS" ]; then
  EXTERNAL_ADDRESS="127.0.0.1"
fi

read -rp "Please enter the database name for archon (default: archondb): " ARCHON_DB_NAME
if [ ! "$ARCHON_DB_NAME" ]; then
  ARCHON_DB_NAME="archondb"
fi

DEFAULT_DB_ADDR="127.0.0.1"
if [ $DOCKER = "y" ]; then
  DEFAULT_DB_ADDR="0.0.0.0"
fi

read -rp "Please enter the database address for archon (default: $DEFAULT_DB_ADDR): " ARCHON_DB_HOST
if [ ! "$ARCHON_DB_HOST" ]; then
  ARCHON_DB_HOST="$DEFAULT_DB_ADDR"
fi

read -rp "Please enter the username for the archon database (default: archonadmin): " ARCHON_DB_USER
if [ ! "$ARCHON_DB_USER" ]; then
  ARCHON_DB_USER="archonadmin"
fi

read -rp "Please enter the password for the archon database (default: psoadminpassword): " ARCHON_DB_PASS
if [ ! "$ARCHON_DB_PASS " ]; then
  ARCHON_DB_PASS="psoadminpassword"
fi

cd "$(dirname "${BASH_SOURCE[0]}")"
SETUP_DIR=$(pwd)
# Move up to the base checkout so that we can build everything.
pushd "$SETUP_DIR/../" > /dev/null

# The user can provide the install location as the first option
# If it's not provided, we'll use a subdirectory of the archon repo.
if [ -z "$1" ]; then
  mkdir -p archon_server
  INSTALL_DIR="$(pwd)/archon_server"
else
  if [ ! -d "$1" ]; then
    mkdir -p "$INSTALL_DIR" || echo "Failed to create installation directory."
  fi
  INSTALL_DIR="$1"
fi

BIN_DIR="$INSTALL_DIR/bin" make build
cd "$INSTALL_DIR"

# Copy all setup files to the server folder.
rsync -r --exclude="*.sh" "$SETUP_DIR"/* .

# Edit default patches directory.
SEARCH='patch_dir: "/usr/local/etc/archon/patches"'
REPLACE="patch_dir: \"$(pwd)/patches\""
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit default parameters directory
SEARCH='parameters_dir: "/usr/local/etc/archon/parameters"'
REPLACE="parameters_dir: \"$(pwd)/parameters\""
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit hostname
SEARCH='hostname: 0.0.0.0'
REPLACE="hostname: $SERVER_IP"
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit external address
SEARCH='external_ip: 127.0.0.1'
REPLACE="external_ip: $EXTERNAL_ADDRESS"
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit certificate location
SEARCH='shipgate_certificate_file: "certificate.pem"'
REPLACE="shipgate_certificate_file: \"$(pwd)/certificate.pem\""
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit key location
SEARCH='ssl_key_file: "key.pem"'
REPLACE="ssl_key_file: \"$(pwd)/key.pem\""
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Edit key location
SEARCH='host: 127.0.0.1'
REPLACE="host: $ARCHON_DB_HOST"
sed_replace "$SEARCH" "$REPLACE" 'config.yaml'

# Docker creates these credentials
if [  "$FIRST_TIME_SETUP" = "y" ] && [ "$DOCKER" = "n" ]; then
  echo "You will be prompted for the database password."
  createdb "$ARCHON_DB_NAME" -h "$ARCHON_DB_HOST"
  psql -d "$ARCHON_DB_NAME" -h "$ARCHON_DB_HOST" -c "CREATE USER $ARCHON_DB_USER WITH ENCRYPTED PASSWORD '$ARCHON_DB_PASS';"
  psql -d "$ARCHON_DB_NAME" -h "$ARCHON_DB_HOST" -c "GRANT ALL ON ALL TABLES IN SCHEMA public TO $ARCHON_DB_USER;"
fi

# This should exist, but let's verify just in case.
if [ ! -d "$INSTALL_DIR"/patches ]; then
  mkdir "$INSTALL_DIR"/patches
  cp -r "$SETUP_DIR"/patches/* "$INSTALL_DIR"/patches/.
fi

echo "Generating certificates..."
./bin/certgen --ip "$SERVER_IP" > /dev/null 2>&1
echo "Done."

if [ "$FIRST_TIME_SETUP" = "y" ]; then
  echo "Adding account..."
  ../bin/account --config . add
  echo "Done."
fi

echo
echo "Archon setup is complete."
echo
echo "If there are patch files you would like the server to verify, please copy them into:"
echo "  $(pwd)/patches"
echo
echo "Please verify the config file has the correct settings before running."
echo "To run the server, execute the following:"
echo "  $(pwd)/bin/archon server --config $(pwd)"
echo
