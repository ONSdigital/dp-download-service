#Feature: Health check
#   *****   Note - this is incompatible with the use of `dphandlers.IdentityWithHTTPClient(clientsidentity.NewWithHealthClient(zc))` in the middleware chain  ******
#   *****   See service.go at about line 173   *******
#  Scenario:
#    When I GET "/health"
#    Then the HTTP status code should be "200"