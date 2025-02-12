#!/bin/bash

timestamp=$(date -Ins)

json_msg=(
"{"
    "\"Foo\": \"bar\""
"}"
)

nats request observations.down.tapir-pop "${json_msg[*]}"
