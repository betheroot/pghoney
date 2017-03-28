# pghoney

A simple Postgres honey pot inspired by [Elastichoney](https://github.com/jordan-wright/elastichoney)

### Getting Started

To install dependencies
`go get ./...`

To run pghoney (currently hardcoded to port 5433)
`go run *.go`

### To Test:
[ ] - hpfeeds

### Initial Release TODO:
[ ] - Work properly with nmap scan
[ ] - Work properly with nmap pgsql-brute
[ ] - Save passwords somewhere ***
[ ] - Utilize command line args for various options
  * interface
  * port
  * hpfeed config
    - host
    - port
    - secret
    - ident?
  * maxBufSize
  * tcpTimeout
  * Users that exist
  * Debug logging
[ ] - Graceful exiting
[ ] - Clean up logging
[ ] - Write unit tests
[ ] - Write integration tests using nmap + psql

### TODO
[ ] - Support md5 authentication
