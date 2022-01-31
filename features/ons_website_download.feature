Feature: ONS Public Website Download files

    Background:
        Given we are in web mode

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
    And the S3 file "data/populations.csv" with content:
        """
        mark,1
        jon,2
        russ,3
        Ioannis,4
        """
    When I download the file "data/populations.csv"
    Then the HTTP status code should be "200"
    And the headers should be:
        | Content-Type        | application/octet-stream             |
        | Content-Length      | 29                                   |
        | Content-Disposition | attachment; filename=populations.csv |
    And the file content should be:
      """
      mark,1
      jon,2
      russ,3
      Ioannis,4
      """