#!/bin/sh
# check param
if [ "$#" -eq 1 ]; then
    version="$1"
elif [ "$#" -eq 0 ]; then
    version="latest"
else
    echo "Usage: $0 [<version>]"
    exit 1
fi

basedir=`cd $(dirname $0); pwd -P`
echo ${basedir}

# build image
docker build -t imagebuildtool:"$version" -f ${basedir}/Dockerfile .

