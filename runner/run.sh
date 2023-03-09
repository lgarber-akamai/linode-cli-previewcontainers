#!/bin/ash

git clone --branch ${GIT_REPO_BRANCH} ${GIT_REPO_URL} /linode-cli

cd /linode-cli && make install

cd ~ && rm -rf /scripts /linode-cli

printf "Linode API Token:\n> "

read INPUT_TOKEN

LINODE_CLI_TOKEN=$INPUT_TOKEN ash