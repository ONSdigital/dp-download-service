Feature: File event logging in publishing mode

  Background:
    Given the application is in "publishing" mode
    And I am authorised
    And I am identified as "publisher@ons.gov.uk"

  Scenario: File event is logged when downloading in publishing mode
    Given the file "data/published.csv" has the metadata:
      """
      {
        "path": "data/published.csv",
        "is_publishable": true,
        "title": "Published Data",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "PUBLISHED"
      }
      """
    And the file "data/published.csv" is in S3 with content:
      """
      mark,1
      jon,2
      russ,3
      """
    When I GET "/downloads-new/data/published.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"

  Scenario: File download works when file event logging fails
    Given the file "data/test.csv" has the metadata:
      """
      {
        "path": "data/test.csv",
        "is_publishable": true,
        "title": "Test Data",
        "size_in_bytes": 10,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "PUBLISHED"
      }
      """
    And the file "data/test.csv" is in S3 with content:
      """
      test,data
      """
    When I GET "/downloads-new/data/test.csv"
    Then the HTTP status code should be "200"

  Scenario: File event logged for uploaded file in publishing mode
    Given the file "data/uploaded.csv" has the metadata:
      """
      {
        "path": "data/uploaded.csv",
        "is_publishable": true,
        "title": "Uploaded Data",
        "size_in_bytes": 15,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "UPLOADED"
      }
      """
    And the file "data/uploaded.csv" is in S3 with content:
      """
      test,upload
      """
    When I GET "/downloads-new/data/uploaded.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"