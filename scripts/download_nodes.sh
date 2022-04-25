#!/bin/bash

kubectl get no -o wide -l node-role.kubernetes.io/includedRole=,\
node-role.kubernetes.io/excluededRole!= \
-o json > ./testdata/nodes.json