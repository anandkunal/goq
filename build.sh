#!/usr/bin/env bash

go build -o goq by_queued_item.go journal_reader.go journal_writer.go queued_item.go server.go