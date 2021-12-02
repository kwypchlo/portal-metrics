# nginx-log-filter

This folder contains code for a binary called 'filter', which will parse an
nginx access.log and split it into a bunch of significantly smaller logs that
contain more easily parsed information. These smaller logs are then used
individually in different operations, such as scanning a history of uploads to
see if there are any ip addresses that should be banned, or scanning a history
of uploads to count the number of unique IP addresses that visited the server
each day.
