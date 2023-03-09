#!/bin/ash

git clone --branch ${GIT_REPO_BRANCH} ${GIT_REPO_URL} /linode-cli

cd /linode-cli && make install

cd ~ && rm -rf /scripts /linode-cli

ash