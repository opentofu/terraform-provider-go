go build

dest=~/.terraform.d/plugins/terraform.local/local/go/0.0.1/darwin_arm64/terraform-provider-go_v0.0.1
mkdir -p $(dirname $dest)

cp terraform-provider-go $dest

rm .terraform* -r
tofu init -reconfigure
tofu plan
