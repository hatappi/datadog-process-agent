#!/usr/bin/env bash
curl --user ${CIRCLE_TOKEN}: \
       --request POST \
       --form revision=f075b7961137ad8fef27d4a9e35810fa8611c5a3\
       --form config=@config.yml \
       --form notify=false \
       https://circleci.com/api/v1.1/project/github/DataDog/datadog-process-agent/tree/sunhay/add-bpf-based-network-check
