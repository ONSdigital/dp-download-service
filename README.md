# dp-download-service (proof of concept)

## How to test the POC

(Disclaimer: You will need a fully runnable local environment)

- Create an RSA Private Key file in your `$HOME` by running:

`openssl genrsa  -out private.pem 1024`

- Set the environment variable:

`export RSA_PRIVATE_KEY=$(cat $HOME/private.pem)`

- Run the dataset api on branch poc/download-service
- Run the dp-download-service poc:

 `go run main.go`

- Import and publish a new version of a dataset - see [dp-import](https://github.com/ONSdigital/dp-import). Please note that florence will need to be run with encryption disabled. (`export ENCRYPTION_DISABLED=true`)
- Edit the csv download for the version in mongodb to include a new "public" field, which matches current url field, and update the url to point to the download service. E.g:

```json
 "csv" : {
            "url" : "http://localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv",
            "size" : "2439056",
            "public" : "https://csv-exported.s3.eu-west-1.amazonaws.com/0489f298-5324-4db7-9efc-6cc0beb4e7cf.csv"
        }
```

- Verify that when you request the csv download from the download service, then you are redirected to the public link. You can test this through curl and would expect a 302 status code with a location header to the public csv:

`curl -v localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv`

- To test the private link you must now switch florence encryption on (`export ENCRYPTION_DISABLED=false`) and restart florence.
- Upload a file through the [florence UI](http://localhost:8081/florence/uploads/data). There is no need to click the `save and continue` button - just copy the filename from the s3 url on the screen once it has uploaded. An example file is available [here](https://github.com/ONSdigital/dp-web-tests/blob/cmd-develop/testdata/cpicoicoptest.csv)
- Again, edit the mongo document, this time removing the public field and replacing it with a private field, with a value corresponding to the uploaded file:

```json
 "csv" : {
            "url" : "http://localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv",
            "size" : "2439056",
            "private" : "2470609-cpicoicoptestcsv"
        }
```

- This time, instead of being redirected to a public url, you should see that the file is decrypted and streamed back in the http response body:

`curl -v localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv > version.csv`

- Finally, change the state of your version in mongo to `associated`. Now when you attempt the above request, you should be returned a `Not found` http status, however when you authenticate your request, you should once again be able to download the file:

`curl -v localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/2.csv -H 'Internal-Token: AL0108EA-825D-411C-9B1D-41EF7727F46' > version.csv`
