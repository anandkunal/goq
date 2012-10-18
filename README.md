# goq

goq is a fast and simple queue written in Go.


## Features

- Communication over TCP (handles thousands of operations/second)
- Command-based text protocol (easy to write client libraries)
- Minimal API (`enq`, `dew`, `stats`, `version`, and `quit`)
- Basic configuration (`port`, `max_memory_size`, and `journals`)
- Operations are journaled for persistence (serialization via `MsgPack`)
- Journals are health checked during startup, allowing for safe recovery


## Installation

1. Download or clone this repository to your system.
2. `cd` into the repository and execute `./build.sh`. Note, this will create a binary called `goq`. This process will fail unless you have already installed the Go tools. If you continue to have issues with this process, please file a GitHub issue.
3. After successfully creating the binary, read about how to configure and run your first instance.


## Configuration

There are only three configuration parameters needed to run goq. Instead of managing a specific file, these are simply binary arguments/flags.

These three parameters are:

- **[p]ort** - the port that this instance of goq should listen on. Note, you can have multiple instances running on a single server.

- **[m]emory** - the maximum number of bytes that goq should store in memory before spawning additional journals. Tune this value to the amount of available RAM. Note, journals will be sized relative to this value.

- **[j]ournals** - the path to the journals directory. Make sure that this directory already exists. Create/adjust the necessary permissions to ensure that goq can read/write files in this directory.

By default goq listen on all addresses. If you want to listen on only one address use this parameter:

- **[a]ddress** - the address that this instance of goq should listen on.


## Initialization

Now that you've created a binary and read about the configuration parameters above, you are ready to fire up an instance. To do so, go ahead and execute the following command:

	./goq -port=11311 -memory=100000 -journals=/var/log/goq/

Additional permissions may be required if you try to bind to ports `80` and `443`.

After you execute the command, you should see your terminal contain the following log information:

	2012/10/15 10:34:20 Memory bytes: 67108864
	2012/10/15 10:34:20 Listening on port: 11311
	2012/10/15 10:34:20 Journals directory: /var/log/goq/
	2012/10/15 10:34:20 Spawning journal writer: /var/log/goq/1350322460.log
	2012/10/15 10:34:20 Ready...

This informs you that a journal was created in the specified directory. Learn about goq's API from the documentation below.


## API

The fastest way to understand the goq API is to execute simple commands via the terminal. Instead of using a programming language or library, we'll just use `nc`.

If you went through the above instructions (created a binary, executed the binary on port 11311) go ahead and type in the following command in your terminal:

	nc localhost 11311
	
If you take a look at the goq log, you should see something like:

	2012/10/15 10:35:46 Connected to [::1]:49256

goq is reporting that it has connected to a client with that particular address (local).

Now that you have connected, let's walk through the entire API. There are only 5 API methods, so this should be quick.

Note, this API uses a text protocol where commands are terminated by newlines. This will be important for enqueuing data.

### Enqueue

To enqueue data, use the **enq** command. goq accepts data in any format: plain text, JSON, MsgPack, Thrift, ProtoBuf, etc. The only caveat is that your data cannot contain any newlines (`\n`). A quick and easy way to get around this is to simply encode the data into a form that neutralizes newlines, for instance Base64.

To enqueue something, execute the following command from your client:

	enq I am the first item!

Go ahead and enqueue more items using the above command.

### Dequeue

To dequeue data, use the **deq** command. Execute the following from your client:

	deq

If you followed the first enqueue command from above, you should see:

	I am the first item!

Your data is returned in the exact format that it was enqueued. If you enqueued anything else, try dequeueing again. If you call dequeue on an empty queue, you will receive `NIL`.

### Stats

To see statistics about the current goq instance, use the **stats** command.

Execute the following from your client:

	stats

You should receive a JSON dump of the server's current statistics in a format similar to the following:

	{"memory_count":0,"total_count":0,"memory_bytes":0"current_state":1}

Your values may be different from above if you've enqueued or dequeued other items.

### Version

To see the current goq version, use the **version** command.

Execute the following from your client:

	version
	
As of the most recent version, you should see:

	1.0.0

This should be parsed as a string. The version number will become increasingly important as features are introduced.

### Quit

To disconnect from goq, use the **quit** command.

Execute the following from your client:

	quit
	
You should no longer be connected to the goq instance.


## Journals (Internals)

You do not need to care how journals work to use goq. This section is for people that want to learn more about how goq persists data to disk and how journals are replayed.

Let's say you create a new goq instance and execute the following operations:

	enq Hello, Journals!
	deq

From the goq instance, you should see the path of the journal that was created. Go ahead and `cat` that file. Log names follow the format: `[epoch].log`. You should see something like:

	1350355820356844000?Hello, Journals!
	1350355820356844000

The first line represents an enqueued item. An enqueue is journaled with a timestamp (nanoseconds) along with a serialized representation (MsgPack) of the data. The timestamp is the identifier of the operation and is used for dequeues.

The second line represents a dequeue. Unlike the line above, a dequeue only requires the identifier of a previously enqueued item. No further data or timestamp is necessary.

New journals are spawned when the amount of bytes for the enqueued items exceeds the configured parameter. Enqueues and dequeues belong to the same journals. Journals are append only.

If we restart this instance, goq replay all of the journals to achieve consistency. You can restore a goq instance just from its journals directory. Please raise an issue if you come across an issue replaying transactions.


## Contributions

goq was developed by [Kunal Anand][0].


## License

This code is completely free under the MIT License: [http://mit-license.org/][2].


[0]: https://twitter.com/ka
[2]: http://mit-license.org/
