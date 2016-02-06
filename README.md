
<p align="center"><img src="http://i.imgur.com/160ZjLq.png"></p>

Olympus is a personal storage platform, written in pure Go, using the graph database [Cayley](https://github.com/google/cayley) as its metadata store. It supports de-duplication by storing data in 1Mb chunks, and associating the hashes of those chunks with files in the graph. Olympus is architected for speed and simplicity, with a simple API inspired by Unix filesystem commands. 

Olympus makes use of a monorepo structure for maximum code reuse. Client and server code use the same model objects, and communicate with each other using Go's wire encoding format, [gob](https://golang.org/pkg/encoding/gob/).

Currently in pre-beta, Olympus is nearly fully-implemented on the server-side, but currently only supports one client, a cli, which itself only supports a handful of filesystem operations, such as: 
 - `ls`
 - `mkdir`
 - `pwd`
 - `cd`
 - `rm`

You can run Olympus on any machine on your local network. Client/server discovery happens automatically via UDP when you start your client.

To run the tests:
```sh
$ make test # Or make testcover for test coverage
```

To install Olympus server:
```sh
$ make && make install # Installs Olympus to /usr/local/bin and rus it as daemon
```


And to install Olympus cli:
```sh
$ make && make install-cli
$ olympus-cli # To run the client
```

Data and config files are, by default, stored in a the current users's home directory under `.olympus/`. To specify an alternative location, set the environment variable `OLYMPUS_HOME` to another path before installing.

## Coming Soon
 - Mobile and Web clients
 - Desktop agent
 - File streaming API
 - HTTP/2 Support
 - Expanded runtime configuration options
 - Remote instance support
