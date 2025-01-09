#!/bin/bash

printf "Testing rate limit with http request, total 5100 requests in 10 seconds. Should expect 5000 requests with status 2xx and 100 requests non 2xx response status \n\n"

# -c 1: Simulates 1 concurrent connections.
# -d 10: Duration of the test in seconds (10 seconds).
# -r 510: Sends 510 requests per second
# Total 5100 requests in 10 seconds. Should expect 5000 requests with status 2xx and 100 requests non 2xx response status
npx autocannon -c 1 -d 10 -r 510 http://localhost:8080/
