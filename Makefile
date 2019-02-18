
setup:
	export GO111MODULE=on
	go mod vendor

deploy:
	gcloud functions deploy coords --runtime go111 --trigger-http --entry-point "GetLocation"

