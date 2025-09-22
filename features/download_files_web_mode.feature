Feature: Download files feature - web

  Background:
    Given the application is in "web" mode
    And I am authorised
    And I am identified as "dave@ons.gov.uk"

  Scenario: File is published and downloaded successfully
    Given the file "data/populations.csv" has the metadata:
      """
      {
        "path": "data/populations.csv",
        "is_publishable": true,
        "collection_id": "1234-asdfg-54321-qwerty",
        "title": "The number of people",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "PUBLISHED"
      }
      """
    And the file "data/populations.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "301"
    And the response header "Content-Disposition" should be "attachment; filename=populations.csv"

  Scenario: File is not uploaded and not published returns 404
    Given the file "data/missing.csv" has the metadata:
      """
      {
        "path": "data/missing.csv",
        "is_publishable": true,
        "collection_id": "1234-asdfg-54321-qwerty",
        "title": "Missing file",
        "size_in_bytes": 0,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "CREATED"
      }
      """
    And the file "data/missing.csv" is not present in S3
    When I download the file "data/missing.csv"
    Then the HTTP status code should be "404"
    And I should receive the following JSON response:
      """
      file not registered
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
    When I download the file "data/unpublished.csv"
    Then the HTTP status code should be "404"
    
  Scenario:File is uploaded but collection is published and file is downloaded
    Given the file "data/collection-published.csv" has the metadata:
      """
      {
        "path": "data/collection-published.csv",
        "is_publishable": true,
        "collection_id": "collection-published-1234",
        "title": "Collection published file",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "UPLOADED"
      }
      """
    And the file "data/collection-published.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    And the collection "collection-published-1234" is marked as PUBLISHED
    When I download the file "data/collection-published.csv"
    Then the HTTP status code should be "200"
    And the response header "Content-Disposition" should be "attachment; filename=collection-published.csv"
