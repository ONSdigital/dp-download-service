Feature: ONS Public Website Download files

  Background:
    Given the application is in "web" mode

  Scenario: Download a file that has been published
    Given the file "data/populations.csv" metadata:
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
    And the file "data/populations.csv" is encrypted in S3 with content:
        """
        mark,1
        jon,2
        russ,3
        Ioannis,4
        """
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "200"
    And the headers should be:
      | Content-Type        | text/csv                             |
      | Content-Length      | 29                                   |
      | Content-Disposition | attachment; filename=populations.csv |
    And the file content should be:
      """
      mark,1
      jon,2
      russ,3
      Ioannis,4
      """

  Scenario: Trying to download a file that has not been uploaded yet
    Given the file "data/populations.csv" has not been uploaded
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "404"

  Scenario: ONS previewer requests data-file that has been uploaded but not yet published
    Given the file "data/populations.csv" has been uploaded
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
          "state": "UPLOADED"
        }
        """
    And the file "data/populations.csv" is encrypted in S3 with content:
        """
        mark,1
        """
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "404"

  Scenario: ONS previewer requests data-file that has been created but not yet uploaded
    Given the file "data/populations.csv" has been uploaded
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
          "state": "UPLOADED"
        }
        """
    And the file "data/populations.csv" is encrypted in S3 with content:
        """
        mark,1
        """
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "404"

  Scenario: Redirecting public to decrypted bucket when file is published & decrypted
    Given the file "data/populations.csv" has been uploaded
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
          "state": "DECRYPTED"
        }
        """
    And the file "data/populations.csv" is encrypted in S3 with content:
        """
        mark,1
        """
    When I download the file "data/populations.csv"
    Then I should be redirected to "http://public-bucket.com/data/populations.csv"