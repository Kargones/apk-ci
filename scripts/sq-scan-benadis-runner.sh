#!/bin/bash
cd /root/r/benadis-runner/
/tmp/4del/scanner/temp/sonar-scanner-4141875950/bin/sonar-scanner \
  -Dsonar.projectKey=benadis-runner \
  -Dsonar.sources=. \
  -Dsonar.host.url=http://sq.apkholding.ru:9000 \
  -Dsonar.token=sqp_ddd64c75da816ba00cd9c56c4c503019203bbc76