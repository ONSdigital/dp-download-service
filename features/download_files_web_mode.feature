Feature: Download files feature - web

  Background:
    Given the application is in "web" mode

  Scenario: File is published and downloaded successfully
    Given the file "data/return301.csv" has the metadata:
      """
      {
        "path": "data/return301.csv",
        "is_publishable": true,
        "collection_id": "1234-asdfg-54321-qwerty",
        "title": "The number of people",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "MOVED"
      }
      """
    And the file "data/return301.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    When I GET "/downloads/files/data/return301.csv"
    Then the HTTP status code should be "301"
    And the response header "Location" should be "http://public-bucket.com/data/return301.csv"
    And no file event should be logged

  Scenario: File is not uploaded and not published returns 404
    Given the file "data/missing.csv" has not been uploaded
    When I GET "/downloads/files/data/missing.csv"
    Then I should receive the following JSON response with status "404":
      """
      {
        "errors": [
          {
            "code": "FileNotRegistered",
            "description": "file not registered"
          }
        ]
      }
      """

  Scenario: File is uploaded but not published returns 404
    Given the file "data/unpublished.csv" has the metadata:
      """
      {
        "path": "data/unpublished.csv",
        "is_publishable": true,
        "collection_id": "1234-asdfg-54321-qwerty",
        "title": "The number of people",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "UPLOADED"
      }
      """
    And the file "data/unpublished.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "404"
    
  Scenario: File is uploaded, collection is published and file is downloaded
    Given the file "data/published.csv" has the metadata:
      """
      {
        "path": "data/published.csv",
        "is_publishable": true,
        "collection_id": "collection-published-1234",
        "title": "Collection published file",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "MOVED"
      }
      """
    And the file "data/published.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    And the collection "collection-published-1234" is marked as PUBLISHED
    When I GET "/downloads/files/data/published.csv"
    Then the HTTP status code should be "200"
    And the response header "Content-Disposition" should be "attachment; filename=published.csv"
    And no file event should be logged
