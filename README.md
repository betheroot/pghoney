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
- [ ] Work properly with nmap pgsql-brute
  * maxBufSize
  * tcpTimeout
- [ ] Create deploy script within fflemming's fork of mhn
- [ ] Support proper error for "cancelling" a query (12345678, very similar to SSL request)
- [ ] Support mechanism for saving passwords in a seperate database.
- [ ] Don't hardcode the md5 salt

### TODO's
- [ ] Write integration tests using nmap + psql
- [ ] Write integration tests using github.com/lib/pq
