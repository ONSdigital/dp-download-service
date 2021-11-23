module github.com/ONSdigital/dp-download-service

go 1.17

replace github.com/coreos/etcd => github.com/coreos/etcd v3.3.24+incompatible
replace github.com/dgrijalva/jwt-go => github.com/dgrijalva/jwt-go v3.2.1-0.20210802184156-9742bd7fca1c+incompatible

require (
	github.com/ONSdigital/dp-api-clients-go/v2 v2.1.14
	github.com/ONSdigital/dp-component-test v0.6.0
	github.com/ONSdigital/dp-healthcheck v1.1.0
	github.com/ONSdigital/dp-net/v2 v2.2.0-beta
	github.com/ONSdigital/dp-s3 v1.7.0
	github.com/ONSdigital/dp-vault v1.1.1
	github.com/ONSdigital/log.go/v2 v2.0.9
	github.com/aws/aws-sdk-go v1.40.37
	github.com/cucumber/godog v0.12.2
	github.com/golang/mock v1.4.4
	github.com/gorilla/handlers v1.4.2
	github.com/gorilla/mux v1.8.0
	github.com/justinas/alice v1.2.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/maxcnunes/httpfake v1.2.4
	github.com/smartystreets/goconvey v1.6.4
	github.com/stretchr/testify v1.7.0
)

require (
	github.com/ONSdigital/dp-api-clients-go v1.41.1 // indirect
	github.com/ONSdigital/dp-mongodb-in-memory v1.0.0 // indirect
	github.com/ONSdigital/dp-mongodb/v3 v3.0.0-beta.1 // indirect
	github.com/ONSdigital/dp-net v1.0.12 // indirect
	github.com/ONSdigital/log.go v1.0.1 // indirect
	github.com/ONSdigital/s3crypto v0.0.0-20180725145621-f8943119a487 // indirect
	github.com/cucumber/gherkin-go/v19 v19.0.3 // indirect
	github.com/cucumber/messages-go/v16 v16.0.1 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/fatih/color v1.12.0 // indirect
	github.com/go-stack/stack v1.8.0 // indirect
	github.com/gofrs/uuid v4.0.0+incompatible // indirect
	github.com/golang/snappy v0.0.2 // indirect
	github.com/gopherjs/gopherjs v0.0.0-20210202160940-bed99a852dfe // indirect
	github.com/hashicorp/errwrap v1.0.0 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.1 // indirect
	github.com/hashicorp/go-hclog v0.12.0 // indirect
	github.com/hashicorp/go-immutable-radix v1.3.0 // indirect
	github.com/hashicorp/go-memdb v1.3.0 // indirect
	github.com/hashicorp/go-multierror v1.0.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.6.2 // indirect
	github.com/hashicorp/go-rootcerts v1.0.1 // indirect
	github.com/hashicorp/go-sockaddr v1.0.2 // indirect
	github.com/hashicorp/golang-lru v0.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/api v1.0.5-0.20200527182800-ad90e0b39d2f // indirect
	github.com/hashicorp/vault/sdk v0.1.14-0.20200514144402-4bfac290c352 // indirect
	github.com/hokaccha/go-prettyjson v0.0.0-20210113012101-fb4e108d2519 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/jtolds/gls v4.20.0+incompatible // indirect
	github.com/klauspost/compress v1.11.1 // indirect
	github.com/mattn/go-colorable v0.1.8 // indirect
	github.com/mattn/go-isatty v0.0.13 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/mapstructure v1.2.2 // indirect
	github.com/pierrec/lz4 v2.0.5+incompatible // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/ryanuber/go-glob v1.0.0 // indirect
	github.com/smartystreets/assertions v1.2.0 // indirect
	github.com/spf13/afero v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/square/mongo-lock v0.0.0-20201208161834-4db518ed7fb2 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.0.2 // indirect
	github.com/xdg-go/stringprep v1.0.2 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.mongodb.org/mongo-driver v1.7.1 // indirect
	golang.org/x/crypto v0.0.0-20210314154223-e6e6c4f2bb5b // indirect
	golang.org/x/net v0.0.0-20210805182204-aaa1db679c0d // indirect
	golang.org/x/sync v0.0.0-20201020160332-67f06af15bc9 // indirect
	golang.org/x/sys v0.0.0-20210809222454-d867a43fc93e // indirect
	golang.org/x/text v0.3.6 // indirect
	golang.org/x/time v0.0.0-20190308202827-9d24e82272b4 // indirect
	gopkg.in/square/go-jose.v2 v2.3.1 // indirect
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
)
