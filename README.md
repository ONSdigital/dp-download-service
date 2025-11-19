# DP Download Service

## Introduction

The Download Service is part of the [Static Files System](https://github.com/ONSdigital/dp-static-files-compose).
This service is responsible for storing the metadata and state of files.

The service uses the [Files API](https://github.com/ONSdigital/dp-files-api) to retrieve a files metadata, whether it
is web or publishing mode and the users role to work out how it should respond to the request to download a file.

If the download service is in publishing mode and the user is allowed to review a file then the file is viewable at any
time as long as the file state is not CREATED (File is still being uploaded)

In web mode the services respond differently depending on the state of the file. The table below show the HTTP response
for each state and why the Download Service responds in such a way.

| context                      | State       | HTTP Response           | Notes                                                      |
|------------------------------|-------------|-------------------------|------------------------------------------------------------|
| any                          | CREATED     | 404 - Not Found         | File is being uploaded do not expose file exists to public |
| auth user in publishing mode | UPLOADED    | 200 - OK                | File is previewable - stream content from S3               |
| web (anon) user in web mode  | UPLOADED    | 404 - Not Found         | File is being reviewed do not expose file exists to public |
| any                          | MOVED       | 301 - Moved Permanently | File is moved - redirect request to public location        | 
| any                          | PUBLISHED   | 200 - OK                | File is published - stream content from S3                 |

## Installation

Service is authenticated against zebedee, one can run [dp-auth-api-stub](https://github.com/ONSdigital/dp-auth-api-stub)
to mimic service identity check in zebedee.

### AWS credentials

The app uses the default provider chain. When running locally this typically means they are provided by
the `~/.aws/credentials` file. Alternatively you can inject the credentials via environment variables as described in
the configuration section

## Healthcheck

The endpoint `/healthcheck` checks the health of the dataset api and returns one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

## Configuration

| Environment variable         | Default                              | Description                                                                                      |
|------------------------------|--------------------------------------|--------------------------------------------------------------------------------------------------|
| BIND_ADDR                    | :23600                               | The host and port to bind to                                                                     |
| BUCKET_NAME                  | "csv-exported"                       | The s3 bucket to retrieve files from                                                             |
| DATASET_API_URL              | http://localhost:22000               | The dataset api url                                                                              |
| DATASET_AUTH_TOKEN           | FD0108EA-825D-411C-9B1D-41EF7727F465 | The dataset auth token                                                                           |
| DOWNLOAD_SERVICE_TOKEN       | QB0108EZ-825D-412C-9B1D-41EF7747F462 | The token to request public/private links from dataset api                                       |
| FILTER_API_URL               | http://localhost:22100               | The filter api url                                                                               |
| IMAGE_API_URL                | http://localhost:24700               | The image api url                                                                                |
| FILES_API_URL                | http://localhost:26900               | The image api url                                                                                |
| SECRET_KEY                   | -                                    | A secret key used authentication                                                                 |
| GRACEFUL_SHUTDOWN_TIMEOUT    | 5s                                   | The graceful shutdown timeout in time duration string format                                     |
| HEALTHCHECK_INTERVAL         | 30s                                  | The period of time between health checks                                                         |
| HEALTHCHECK_CRITICAL_TIMEOUT | 90s                                  | The period of time after which failing checks will result in critical global check status        |
| OTEL_BATCH_TIMEOUT           | 5s                                   | Interval between pushes to OT Collector                                                          |
| OTEL_EXPORTER_OTLP_ENDPOINT  | http://localhost:4317                | URL for OpenTelemetry endpoint                                                                   |
| OTEL_SERVICE_NAME            | "dp-download-service"                | Service name to report to telemetry tools                                                        |
| OTEL_ENABLED                 | false                                | Feature flag to enable OpenTelemetry                                                             |
| SERVICE_AUTH_TOKEN           | c60198e9-1864-4b68-ad0b-1e858e5b46a4 | The service auth token for the download service                                                  |
| ZEBEDEE_URL                  | http://localhost:8082                | The URL for zebedee                                                                              |
| AWS_REGION                   | -                                    | The AWS access key credential                                                                    |
| AWS_ACCESS_KEY_ID            | -                                    | The AWS access key credential                                                                    |
| AWS_SECRET_ACCESS_KEY        | -                                    | The AWS secret key credential                                                                    |
| IS_PUBLISHING                | true                                 | Determines if the instance is publishing or not                                                  |

## API Client 

There is an [API Client](https://github.com/ONSdigital/dp-api-clients-go/tree/main/download) for the Download API this is part
of [dp-api-clients-go](https://github.com/ONSdigital/dp-api-clients-go) package.

## Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

## License

Copyright © 2022, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details.