### Build and deploy

1. Follow the basic setup here: https://cloud.google.com/run/docs/quickstarts/build-and-deploy/deploy-go-service

2. Amend `cloudbuild.yaml` with `gcr.io/<PROJECT_ID>/<YOUR_SERVICE_NAME>`

3. Run `gcloud builds submit` at the project root

4. Navigate to Cloud Run on the GC Console and deploy the submitted build

### Local development

1. `brew install sqlc` and then `sqlc generate` to generate db methods and models