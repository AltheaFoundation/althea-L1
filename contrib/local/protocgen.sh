#!/usr/bin/env bash

set -eo pipefail

if [ -z $GOPATH ]; then
	echo "GOPATH not set!"
	exit 1
fi

if [[ $PATH != *"$GOPATH/bin"* ]]; then
	echo "GOPATH/bin must be added to PATH"
	exit 1
fi

cd proto

proto_dirs=$(find ./althea -path -prune -o -name '*.proto' -print0 | xargs -0 -n1 dirname | sort | uniq)
for dir in $proto_dirs; do
  for file in $(find "${dir}" -maxdepth 1 -name '*.proto'); do
    if grep go_package $file &>/dev/null; then
      echo "Generating gogo proto code for $file"
      buf generate $file --template buf.gen.gogo.yaml
    fi
  done
done

cd ..

# move proto files to the right places
cp -r github.com/AltheaFoundation/althea-L1/* ./
rm -rf github.com
