build:
	go build .
.PHONY: build

deploy: build
	gcloud functions deploy TestPlatformResultsIndexJobsByNamespace \
		--project openshift-gce-devel --runtime go121 \
		--service-account search-index-gcs-writer@openshift-gce-devel.iam.gserviceaccount.com \
		--memory 128MB --timeout=15s --max-instances=10 \
		--trigger-resource test-platform-results --trigger-event google.storage.object.finalize
.PHONY: deploy-functions

delete:
	gcloud functions delete TestPlatformResultsIndexJobsByNamespace \
		--project openshift-gce-devel
.PHONY: delete-functions
