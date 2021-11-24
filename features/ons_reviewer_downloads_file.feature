Feature: Download preview feature

  Background:
    Given I am authorised
    And I am identified as "dave@ons.gov.uk"

  Scenario: Return the dataset when it exists in collection
    When I request to download the file "not-a-file.csv"
    Then the HTTP status code should be "404"

  Scenario: ONS previewer requests data-file that has been uploaded but not yet decrypted
    Given the file "cpih01-time-series-v5.csv" has been uploaded
    But is not yet published
    When I request to download the file "cpih01-time-series-v5.csv"
    Then I should receive the private file "cpih01-time-series-v5.csv"

#  Scenario: ONS previewer requests data-file that has been uploaded and published
#    Given the file "cpih01-time-series-v5.csv" has been uploaded
#    And has been published
#    When I request to download the file "cpih01-time-series-v5.csv"
#    Then I should redirected to the public file "cpih01-time-series-v5.csv"
