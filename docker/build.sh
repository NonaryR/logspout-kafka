#!/bin/sh
set -e
echo -e "https://mirror.tuna.tsinghua.edu.cn/alpine/v3.5/main\nhttps://mirror.tuna.tsinghua.edu.cn/alpine/v3.5/community" > /etc/apk/repositories
apk add --update go git mercurial build-base ca-certificates
