#!/bin/sh

HTTP_LOAD_PARALLEL=10 HTTP_LOAD_SECONDS=60 TARGET=127.0.0.1:80 NODE_PATH=lib node bench.js team01 standalone
