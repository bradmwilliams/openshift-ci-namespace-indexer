build:
	go build .
.PHONY: build

deploy: build
	gcloud functions deploy IndexJobsByNamespace \
		--project openshift-gce-devel --runtime go120 \
		--service-account search-index-gcs-writer@openshift-gce-devel.iam.gserviceaccount.com \
		--memory 128MB --timeout=15s --max-instances=10 \
		--trigger-resource origin-ci-test --trigger-event google.storage.object.finalize
.PHONY: deploy-functions

delete:
	gcloud functions delete IndexJobsByNamespace \
		--project openshift-gce-devel
.PHONY: delete-functions
