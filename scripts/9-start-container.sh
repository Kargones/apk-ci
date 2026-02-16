#!/bin/bash
# docker run h -p 2222:22 git.apkholding.ru/xor/ar-edt:5.5.25.1 /usr/sbin/sshd -D
docker run -d \
  --name ar-edt \
  -p 2222:22 \
  git.apkholding.ru/xor/ar-edt:5.7.28.1
