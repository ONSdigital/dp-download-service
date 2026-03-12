Feature: Download preview feature - publishing

  Background:
    Given the application is in "publishing" mode

  Scenario: File is published and downloaded successfully
    Given the file "data/published.csv" has the metadata:
      """
      {
        "path": "data/published.csv",
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
    And the file "data/published.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    And I am a publisher user
    When I GET "/downloads/files/data/published.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"
    And the response header "Content-Disposition" should be "attachment; filename=published.csv"
    And a file event with action "READ" and resource "data/published.csv" should be created by user "janedoe@example.com"
  
  Scenario: File is published and downloaded successfully (With only an access_token cookie)
    Given the file "data/published.csv" has the metadata:
      """
      {
        "path": "data/published.csv",
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
    And the file "data/published.csv" is in S3 with content:
      """
      mark,1
      russ,2
      dan,3
      saul,3.5
      brian,4
      jon,5
      """
    And I am an admin user accessing the file through a browser
    When I GET "/downloads/files/data/published.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"
    And the response header "Content-Disposition" should be "attachment; filename=published.csv"
    And a file event with action "READ" and resource "data/published.csv" should be created by user "janedoe@example.com"

  Scenario: File is not uploaded and not published returns 404
    Given the file "data/missing.csv" has not been uploaded
    And I am an admin user
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

  Scenario: File is uploaded but not published and file is downloaded
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
    And I am an admin user
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"
    And the response header "Content-Disposition" should be "attachment; filename=unpublished.csv"
    And a file event with action "READ" and resource "data/unpublished.csv" should be created by user "janedoe@example.com"

  Scenario: File is uploaded but collection is published and file is downloaded
    Given the file "data/published.csv" has the metadata:
      """
      {
        "path": "data/published.csv",
        "is_publishable": true,
        "collection_id": "published-1234",
        "title": "Collection published file",
        "size_in_bytes": 29,
        "type": "text/csv",
        "licence": "OGL v3",
        "licence_url": "http://www.nationalarchives.gov.uk/doc/open-government-licence/version/3/",
        "state": "UPLOADED"
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
    And the collection "published-1234" is marked as PUBLISHED
    And I am an admin user
    When I GET "/downloads/files/data/published.csv"
    Then the HTTP status code should be "200"
    And the response header "Cache-Control" should be "no-cache"
    And the response header "Content-Disposition" should be "attachment; filename=published.csv"
    And a file event with action "READ" and resource "data/published.csv" should be created by user "janedoe@example.com"

  Scenario: An authorised viewer user requests a file that has been uploaded but not yet published
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
          "state": "UPLOADED",
          "content_item":{
            "dataset_id":"cpih01",
            "edition":"feb-2026"
          }
        }
        """
        And I am a viewer user with permission
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
        Then the HTTP status code should be "200"
        And the response header "Cache-Control" should be "no-cache"
        And the response header "Content-Disposition" should be "attachment; filename=unpublished.csv"
        And a file event with action "READ" and resource "data/unpublished.csv" should be created by user "viewer1@ons.gov.uk"

    Scenario: A viewer user with no permission requests a file that has been uploaded but not yet published
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
          "state": "UPLOADED",
          "content_item":{
            "dataset_id":"cpih01",
            "edition":"feb-2026"
          }
        }
        """
        And I am a viewer user without permission
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
        Then the HTTP status code should be "403"
        And the response header "Cache-Control" should be "no-cache"

     Scenario: A service account requests a file that has been uploaded but not yet published
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
          "state": "UPLOADED",
          "content_item":{
            "dataset_id":"cpih01",
            "edition":"feb-2026"
          }
        }
        """
        And I am identified as "service"
        And I use a service auth token "test-auth-token"
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
        Then the HTTP status code should be "403"
        And the response header "Cache-Control" should be "no-cache"


    Scenario: A request is made for a file that has been uploaded but not yet published and no auth is provided
        Given the file "data/return401.csv" has the metadata:
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
          "state": "UPLOADED",
          "content_item":{
            "dataset_id":"cpih01",
            "edition":"feb-2026"
          }
        }
        """
        And the file "data/return401.csv" is in S3 with content:
        """
        mark,1
        russ,2
        dan,3
        saul,3.5
        brian,4
        jon,5
        """
        When I GET "/downloads-new/data/return401.csv"
        Then the HTTP status code should be "401"
        And the response header "Cache-Control" should be "no-cache"
    
    
  
  
