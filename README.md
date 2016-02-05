
<p align="center"><img src="http://i.imgur.com/6ETS8m5.png"></p>

Olympus is a personal storage platform, written in pure Go, using the graph database [Cayley](https://github.com/google/cayley) as its metadata store. It offers de-duplication by storing data in 1Mb chunks, and associating the hashes of those chunks with files in the graph. Olympus is architected for speed and simplicity, with a simple api inspired by Unix filesystem commands. 

Olympus makes use of a monorepo structure for maximum code reuse. Client and server code use the same model objects, and communicate with each other using Go's wire encoding format, [gob](https://golang.org/pkg/encoding/gob/).

Currently in pre-beta, Olympus is nearly fully-implemented on the server-side, but currently only supports one client, a cli, which itself only supports a handful of filesystem operations, such as: 
 - `ls`
 - `mkdir`
 - `pwd`
 - `cd`

You can run Olympus on any machine on your local network. Client/server discovery happens automatically via UDP when you start your client.

To run the tests:
```sh
$ make test # Or make testcover for test coverage
```

To install Olympus server:
```sh
$ make && make install # Installs and runs the server daemon to /usr/local/bin
```


And to install Olympus cli:
```sh
$ make && make install-cli
$ olympus-cli # To run the client
```
