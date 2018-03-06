# dp-download-service (proof of concept)

## How to test the POC

(Disclaimer: You will need a fully runnable local environment)

- Ensure you have vault started locally:

`brew install vault`
`vault server -dev`

- Run the dataset api on branch poc/download-service
- Create and upload an encrypted file to s3:

`cd scripts`
`make debug`
`cd ..`

- Run the dp-download-service poc:

 `make debug`

- Ensure you have a `published` version available and set the following in mongodb for your version:

```json
 "csv" : {
            "url" : "http://localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv",
            "size" : "2439056",
            "public" : "https://csv-exported.s3.eu-west-1.amazonaws.com/0489f298-5324-4db7-9efc-6cc0beb4e7cf.csv"
        }
```

- Verify that when you request the csv download from the download service, then you are redirected to the public link. You can test this through curl and would expect a 302 status code with a location header to the public csv:

`curl -v localhost:28000/downloads/datasets/931a8a2a-0dc8-42b6-a884-7b6054ed3b68/editions/time-series/versions/1.csv`

- To test the private link, replace your previous mongodb csv download field with this:

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
