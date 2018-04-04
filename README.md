# dp-download-service

An ONS service used to either redirect requests to public-accessible links or stream decrypt non public-accessible links

### Installation

Service is authenticated against zebedee, one can run [dp-auth-api-stub](https://github.com/ONSdigital/dp-auth-api-stub) to mimic service identity check in zebedee.

#### Vault

- Run `brew install vault`
- Run `vault server -dev`

### Healthcheck

The endpoint `/healthcheck` checks the health of vault and the dataset api and returns one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

### Configuration

| Environment variable       | Default                                     | Description
| -------------------------- | --------------------------------------------| -----------
| BIND_ADDR                  | :23600                                      | The host and port to bind to
| BUCKET_NAME                | "csv-exported"                              | The s3 bucket to decrypt files from
| DATASET_API_URL            | http://localhost:22000                      | The dataset api url
| DOWNLOAD_SERVICE_TOKEN     | QB0108EZ-825D-412C-9B1D-41EF7747F462        | The token to request public/private links from dataset api
| SECRET_KEY                 | AL0108EA-825D-411C-9B1D-41EF7727F465        | A secret key used authentication
| DATASET_AUTH_TOKEN         | FD0108EA-825D-411C-9B1D-41EF7727F465        | The host name for the CodeList API
| GRACEFUL_SHUTDOWN_TIMEOUT  | 5s                                          | The graceful shutdown timeout in seconds
| HEALTHCHECK_TIMEOUT        | 60s                                         | The timeout that the healthcheck allows for checked subsystems
| VAULT_ADDR                 | http://localhost:8200                       | The vault address
| VAULT_TOKEN                | -                                           | Vault token required for the client to talk to vault. (Use `make debug` to create a vault token)
| VAULT_PATH                 | secret/shared/psk                           | The path where the psks will be stored in for vault
| SERVICE_AUTH_TOKEN         | c60198e9-1864-4b68-ad0b-1e858e5b46a4        | The service auth token for the download service
| ZEBEDEE_URL                | http://localhost:8082                       | The URL for zebedee

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2018, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details