Feature: Example feature

    Background:
        Given I am authorized

    Scenario: Return the dataset when it exists in collection
        Given the following document exists in the "datasets" collection:
            """
            {
                "_id": "6021403f3a21177b2837d12f",
                "id": "a1b2c3",
                "example_data": "some data"
            }
            """
        When I GET "/downloads/dir/not-a-file.csv"
        Then I should receive the following JSON response with status "200":
            """
            {
                "_id": "6021403f3a21177b2837d12f",
                "msg": "dir/not-a-file.csv could not be found",
                "example_data": "some data"
            }
            """