#!/bin/bash
#
# A script to fix the bug in the controller

SCRIPTDIR=$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )
echo $SCRIPTDIR
FILE_PATH=$SCRIPTDIR"/../config/crds/iter8_v1alpha1_experiment.yaml"
suffix=".original"
line=$(grep 'lastTransitionTime' -n config/crds/iter8_v1alpha1_experiment.yaml | sed 's/:.*//')
line=$(( $line + 5 ))

sed -i$suffix  "${line}s/object/string/" $FILE_PATH
rm $FILE_PATH$suffix
