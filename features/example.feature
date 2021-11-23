Feature: Example feature

    Background:
        Given I am authorised

    Scenario: Return the dataset when it exists in collection
#        Given the following document exists in the "datasets" collection:
#            """
#            {
#                "_id": "6021403f3a21177b2837d12f",
#                "id": "a1b2c3",
#                "example_data": "some data"
#            }
#            """
        Given I am identified as "Dave"
        When I GET "/downloads/dir/not-a-file.csv"
        Then the HTTP status code should be "404"
