# goq

goq is a persistent queue implemented in Go.


## Features

- Communication over HTTP
- RESTful with JSON responses (trivial to write client libraries)
- Minimal API (`enqueue`, `dequeue`, `statistics`, `version`)
- Basic configuration (`address`, `port`, `sync`, and `path`)
- Operations are journaled for persistence via LevelDB
- Database is health checked during startup, allowing for safe recovery


## Dependencies

goq only has one external dependency (LevelDB). Before you start installing anything, go ahead and execute following command:

	`go get github.com/syndtr/goleveldb`


## Installation

1. Download/clone this repository to your system.
2. `cd` into the repository and execute `go build`. Note, this will create a binary called `goq`.
3. After successfully creating the binary, read about how to configure and run your first instance.


## Configuration

There are only a handful of arguments needed to configure goq. Instead of managing a specific file, these are simply binary arguments/flags.

These arguments include:

- **port** - The port that this instance of goq should listen on. Default: 11311.

- **sync** - Synchronize to LevelDB on every write. Default: true.

- **path** - The path to the LevelDB database directory. goq will create this directory if needed. Default: ./db.

goq listen on all addresses by default. If you want to listen on only one address use this parameter:

- **address** - The address that this instance of goq should listen on.


## Initialization

Now that you've created a binary and read about the configuration parameters above, you are ready to fire up an instance. Go ahead and execute the following command:

	./goq -port=11311 -sync=true -journals=/var/log/goq/

After you execute the command, you should see your terminal contain the following log information:

	2014/03/10 13:44:17 Listening on :11311
	2014/03/10 13:44:17 DB Path: /var/log/goq/
	2014/03/10 13:44:17 goq: Starting health check
	2014/03/10 13:44:17 goq: Health check successful
	2014/03/10 13:44:17 Ready...

This informs you that a LevelDB instance was created in the specified directory and that goq is listening on the desired port. Learn about goq's API from the documentation below.


## API

There are only 4 API endpoints of interest, so this should be quick.

### POST /enqueue

To enqueue something, execute the following command from your client:

	POST /enqueue data=>I am the first item!

The only input parameter required is `data`, which is a string. Depending on your use case, you may need to URL encode this parameter.

The response you got back should be:

	{success:true,message:"worked"}

Go ahead and enqueue more items using the above process.

### GET /dequeue

To dequeue data, execute the following from your client:

	GET /dequeue?count=1

Note, if the `count` query parameter isn't specified, goq will only return one item at a time. 

If you followed the first enqueue command from above, you should see:

	{success:true,data:["I am the first item!"],message:"worked"}

Your data is returned in the exact format that it was enqueued. If you enqueued anything else, try dequeueing again. If you call dequeue on an empty queue, you will receive the following: 

	{success:true,data:[],message:"worked"}

### GET /statistics

To see current goq statistics, execute the following from your client:

	GET /statistics

You should see a JSON structure of the server's current statistics in a structure that resembles the following:

	{"enqueues":0,"dequeues":0,"empties":0}

Your values may be different from above if you've enqueued or dequeued other items. `empties` refers to the number of dequeues that were made to an empty queue.

### GET /version

To see the current goq version, execute the following from your client:

	GET /version
	
As of the most recent version, you should see:

	{version:"1.0.0"}

This should be parsed as a string. The version number will become increasingly important as new features are introduced.


## Restarts / Health Checks

When restarted, goq replays all of the items in LevelDB to ensure consistency. You can restore a goq instance just from its LevelDB directory. Please raise an issue if you come across an issue replaying transactions.


## Contributions

goq was developed by [Kunal Anand][0].


## License

This code is completely free under the MIT License: [http://mit-license.org/][2].


[0]: https://twitter.com/ka
[2]: http://mit-license.org/
