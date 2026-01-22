Feature: Download preview feature

    Background:
        Given the application is in "publishing" mode
        And I am authorised
        And I am identified as "dave@ons.gov.uk"

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
        Then the HTTP status code should be "200"
        And the response header "Cache-Control" should be "no-cache"
        And the response header "Content-Disposition" should be "attachment; filename=unpublished.csv"
        And a file event with action "READ" and resource "data/unpublished.csv" should be created by user "dave@ons.gov.uk"

    Scenario: ONS previewer requests data-file with weird characters that has been uploaded but not yet published
        Given the file "data/weird&chars#unpublished.csv" has the metadata:
        """
        {
          "path": "data/weird&chars#unpublished.csv",
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

        And the file "data/weird&chars#unpublished.csv" is in S3 with content:
        """
        mark,1
        russ,2
        dan,3
        saul,3.5
        brian,4
        jon,5
        """
        When I GET "/downloads-new/data/weird&chars#unpublished.csv"
        Then the HTTP status code should be "200"
        And the response header "Cache-Control" should be "no-cache"
        And a file event with action "READ" and resource "data/weird&chars#unpublished.csv" should be created by user "dave@ons.gov.uk"
