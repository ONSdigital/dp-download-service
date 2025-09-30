Feature: ONS Public Website Download files

  Background:
    Given the application is in "web" mode

  Scenario: Download a file that has been published
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
        jon,2
        russ,3
        Ioannis,4
        """
    When I GET "/downloads-new/data/published.csv"
    Then the HTTP status code should be "200"
    And the headers should be:
      | Content-Type        | text/csv                             |
      | Content-Length      | 29                                   |
      | Content-Disposition | attachment; filename=published.csv   |
    And the file content should be:
      """
      mark,1
      jon,2
      russ,3
      Ioannis,4
      """

  Scenario: Download a file with weird characters that has been published
    Given the file "data/weird&chars#published.csv" has the metadata:
        """
        {
          "path": "data/weird&chars#published.csv",
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
    And the file "data/weird&chars#published.csv" is in S3 with content:
        """
        mark,1
        jon,2
        russ,3
        Ioannis,4
        """
    When I GET "/downloads-new/data/weird&chars#published.csv"
    Then the HTTP status code should be "200"
    And the headers should be:
      | Content-Type        | text/csv                                       |
      | Content-Length      | 29                                             |
      | Content-Disposition | attachment; filename=weird&chars#published.csv |
    And the file content should be:
      """
      mark,1
      jon,2
      russ,3
      Ioannis,4
      """

  Scenario: Trying to download a file that has not been uploaded yet
    Given the file "data/missing.csv" has not been uploaded
    When I GET "/downloads-new/data/missing.csv"
    Then the HTTP status code should be "404"
    And the response header "Cache-Control" should be "no-cache"

  Scenario: ONS previewer requests data-file that has been uploaded but not yet published
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
    When I GET "/downloads-new/data/unpublished.csv"
    Then the HTTP status code should be "404"
    And the response header "Cache-Control" should be "no-cache"

  Scenario: Redirecting public to bucket when file is published & moved
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
    When I GET "/downloads-new/data/return301.csv"
    Then I should be redirected to "http://public-bucket.com/data/return301.csv"
    And the response header "Cache-Control" should be "max-age=31536000"
