# pghoney

A simple Postgres honey pot inspired by [Elastichoney](https://github.com/jordan-wright/elastichoney)

### Getting Started

To install dependencies
`go get ./...`

To run pghoney (default is 127.0.0.1:5432)
`go run *.go`

To see the cli help output:
`go run *.go -h`

### To Test:
[ ] - hpfeeds

### Initial Release TODO:
[ ] - Work properly with nmap scan
[ ] - Work properly with nmap pgsql-brute
[ ] - Save passwords somewhere ***
[ ] - Write unit tests
[ ] - Write integration tests using nmap + psql
[ ] - Add command line options for:
  * hpfeed config
    - host
    - port
    - secret
    - ident?
  * maxBufSize
  * tcpTimeout

### TODO
[ ] - Support md5 authentication
