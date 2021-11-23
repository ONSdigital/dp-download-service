# dp-download-service

An ONS service used to either redirect requests to public-accessible links or stream decrypt non public-accessible links

### Installation

Service is authenticated against zebedee, one can run [dp-auth-api-stub](https://github.com/ONSdigital/dp-auth-api-stub) to mimic service identity check in zebedee.

#### Vault

- Run `brew install vault`
- Run `vault server -dev`

#### AWS credentials

The app uses the default provider chain. When running locally this typically means they are provided by the `~/.aws/credentials` file.  Alternatively you can inject the credentials via environment variables as described in the configuration section

### Healthcheck

The endpoint `/healthcheck` checks the health of vault and the dataset api and returns one of:

- success (200, JSON "status": "OK")
- failure (500, JSON "status": "error").

### Configuration

| Environment variable         | Default                                     | Description
| ---------------------------- | --------------------------------------------| -----------
| BIND_ADDR                    | :23600                                      | The host and port to bind to
| ENABLE_MONGO                 | false                                       | Set to true to enable mongo connections
| MONGODB_BIND_ADDR            | localhost:27017                             | The MongoDB bind address
| MONGODB_DATABASE             | _empty_                                     | The MongoDB dataset database
| MONGODB_COLLECTION           | _empty_                                     | MongoDB collection
| MONGODB_USERNAME             | _empty_                                     | MongoDB Username
| MONGODB_PASSWORD             | _empty_                                     | MongoDB Password
| MONGODB_IS_SSL               | false                                       | is SSL enabled for mongo server?
| BUCKET_NAME                  | "csv-exported"                              | The s3 bucket to decrypt files from
| DATASET_API_URL              | http://localhost:22000                      | The dataset api url
| DOWNLOAD_SERVICE_TOKEN       | QB0108EZ-825D-412C-9B1D-41EF7747F462        | The token to request public/private links from dataset api
| FILTER_API_URL               | http://localhost:22100                      | The filter api url
| IMAGE_API_URL                | http://localhost:24700                      | The image api url
| SECRET_KEY                   | -                                           | A secret key used authentication
| DATASET_AUTH_TOKEN           | FD0108EA-825D-411C-9B1D-41EF7727F465        | The dataset auth token
| GRACEFUL_SHUTDOWN_TIMEOUT    | 5s                                          | The graceful shutdown timeout in time duration string format
| HEALTHCHECK_INTERVAL         | 30s                                         | The period of time between health checks
| HEALTHCHECK_CRITICAL_TIMEOUT | 90s                                         | The period of time after which failing checks will result in critical global check status
| VAULT_ADDR                   | http://localhost:8200                       | The vault address
| VAULT_TOKEN                  | -                                           | Vault token required for the client to talk to vault. (Use `make debug` to create a vault token)
| VAULT_PATH                   | secret/shared/psk                           | The path where the psks will be stored in for vault
| SERVICE_AUTH_TOKEN           | c60198e9-1864-4b68-ad0b-1e858e5b46a4        | The service auth token for the download service
| ZEBEDEE_URL                  | http://localhost:8082                       | The URL for zebedee
| AWS_ACCESS_KEY_ID            | -                                           | The AWS access key credential
| AWS_SECRET_ACCESS_KEY        | -                                           | The AWS secret key credential
| IS_PUBLISHING                | true                                        | Determines if the instance is publishing or not
| ENCRYPTION_DISABLED          | false                                       | Determines whether vault is used and whether files are encrypted on S3

### Contributing

See [CONTRIBUTING](CONTRIBUTING.md) for details.

### License

Copyright © 2016-2018, Office for National Statistics (https://www.ons.gov.uk)

Released under MIT license, see [LICENSE](LICENSE.md) for details