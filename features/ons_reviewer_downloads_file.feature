Feature: Download preview feature

    Background:
        Given the application is in "publishing" mode

    Scenario: ONS previewer requests data-file that has been uploaded but not yet published
        Given I am an authorised user
        And the file "data/populations.csv" has been uploaded
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
        Then the HTTP status code should be "200"

    Scenario: ONS previewer requests data-file that has been uploaded but not yet published without a
        Given I am not an authorised user
        And the file "data/populations.csv" has been uploaded
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
        Then the HTTP status code should be "403"
