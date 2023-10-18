# gcp-cloud-trace-otel

In Google Cloud Platform (GCP), Load Balancers play a crucial role in distributing incoming traffic across backend instances to ensure high availability and reliability of your services. However, when GCP Load Balancers handle the initial request from a client, a unique challenge arises regarding trace export.

GCP Load Balancers are responsible for creating the initial request span. This root span, which represents the entire lifecycle of a request, is created within the Load Balancer itself. Unfortunately, due to the architectural design of GCP Load Balancers, this root span cannot be directly exported to external observability platforms, except to Google BigQuery.

This tool is a **Proof of Concept** (PoC) as to how one could export the root span into native OpenTelemetry Protocol (OTLP) format.

## Getting Started

1. Clone the repository to your local machine:

```sh
git clone https://github.com/yourusername/gcp-cloud-trace-otel-converter.git
```

git clone https://github.com/yourusername/gcp-cloud-trace-otel-converter.git

2. Build the tool:

```sh
cd gcp-cloud-trace-otel-converter
go build

```

3. Authenticate to Google Cloud

```sh
gcloud auth login --update-ad
```

4. Export credentials

```sh
export GOOGLE_CREDENTIALS=$(cat /Users/${USER}/.config/gcloud/application_default_credentials.json)
export GOOGLE_APPLICATION_CREDENTIALS=/Users/${USER}/.config/gcloud/application_default_credentials.json
```

5. Export your OTLP endpoint

```sh
export OTEL_EXPORTER_OTLP_ENDPOINT="localhost:4317"
```

6. Run the tool

```sh
./gcp-cloud-trace-otel --project_id=foo-bar-dev-1a2b3c
```
