#!/bin/bash

kubectl get pods -A --field-selector metadata.namespace!=kube-system,\
metadata.namespace!=myawesomenamespace,\
spec.nodeName!=testnode1,\
spec.nodeName!=testnode2\
  -o json > ./testdata/pods.json