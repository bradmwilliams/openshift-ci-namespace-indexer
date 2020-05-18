# openshift-ci-namespace-indexer

GCS function to index the OpenShift CI jobs to their work namespace

The functions operate on origin-ci-test and so must be deployed in the openshift-gce-devel
project. The service account search-index-gcs-writer@openshift-gce-devel.iam.gserviceaccount.com
was created ahead of time and given storage creator/viewer on the bucket. During first deployment
the function should *not* be accessible to external viewers.

Most of the code is from [ci-search-functions](https://github.com/openshift/ci-search-functions)
