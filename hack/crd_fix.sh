#!/bin/bash
#
# A script to fix the bug in the controller-gen

SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
FILE_PATH=$SCRIPTDIR"/../install/helm/iter8-controller/templates/crds/iter8.tools_experiments.yaml"
suffix=".original"
line=$(grep 'lastTransitionTime' -n install/helm/iter8-controller/templates/crds/iter8.tools_experiments.yaml | sed 's/:.*//')
line=$(( $line + 11 ))

sed -i$suffix  "${line}s/object/string/" $FILE_PATH
rm $FILE_PATH$suffix
