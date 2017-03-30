# pghoney

A simple Postgres honey pot inspired by [Elastichoney](https://github.com/jordan-wright/elastichoney)

### Getting Started

To install dependencies
`go get ./...`

To run pghoney (default is 127.0.0.1:5432)
`go run *.go`

To see the cli help output:
`go run *.go -h`

### Initial Release TODO:
[ ] - Work properly with nmap scan
[ ] - Work properly with nmap pgsql-brute
[ ] - Add command line options for:
  * hpfeed config
    - host
    - port
    - secret
    - ident?
  * maxBufSize
  * tcpTimeout
[ ] - Create deploy script within fflemming's fork of mhn

### TODO's
[ ] - Support mechanism for saving passwords in a seperate database.
[ ] - Write integration tests using nmap + psql
[ ] - Write integration tests using github.com/lib/pq
