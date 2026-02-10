Feature: Download preview feature - publishing auth cases

  Background:
    Given the application is in "publishing" mode

  Scenario: Unpublished file with correct permissions group (JWT) returns 200
    Given I am identified as "dave@ons.gov.uk"
    And the file "data/unpublished.csv" has the metadata:
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
    And I use an X Florence user token "valid.jwt.token"
    And the files api allows access
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "200"

  Scenario: Unpublished file with incorrect permissions group (JWT) returns 403
    Given I am identified as "dave@ons.gov.uk"
    And the file "data/unpublished.csv" has the metadata:
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
    And I use an X Florence user token "valid.jwt.token"
    And the files api denies access
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "403"

  Scenario: Unpublished file with no JWT returns 401
    Given I am not identified
    And the file "data/unpublished.csv" has the metadata:
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
    And I am not authorised
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "401"

  Scenario: Unpublished file with invalid JWT returns 401
    Given I am not identified
    And the file "data/unpublished.csv" has the metadata:
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
    And I use an X Florence user token "invalid.jwt.token"
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "401"

  Scenario: Unpublished file with valid service token returns 200
    Given I am not identified
    And the file "data/unpublished.csv" has the metadata:
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
    And I use a service auth token "valid-service"
    When I GET "/downloads/files/data/unpublished.csv"
    Then the HTTP status code should be "200"
