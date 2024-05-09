#!/bin/sh

version="0.0.1"
what_arch=`uname -m`

if [ ${what_arch} == "x86_64" ]
then
    arch="amd64"
elif [ ${what_arch} == "arm64"]
then
    arch=${what_arch}
fi

os=$(uname -s | tr '[:upper:]' '[:lower:]')

go build

dest=~/.terraform.d/plugins/terraform.local/local/go/${version}/${os}_${arch}/terraform-provider-go_v${version}
mkdir -p $(dirname $dest)

cp terraform-provider-go $dest

rm -r .terraform*
tofu init -reconfigure
tofu plan
