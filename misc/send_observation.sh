#!/bin/bash

timestamp=$(date -Ins)

json_msg=(
"{"
    "\"src_name\": \"dns-tapir\","
    "\"creator\": \"\","
    "\"msg_type\": \"observation\","
    "\"list_type\": \"doubtlist\","
    "\"added\": ["
        "{"
            "\"name\": \"$1\","
            "\"time_added\": \"$timestamp\","
            "\"ttl\": 3600,"
            "\"tag_mask\": $2,"
            "\"extended_tags\": []"
        "}"
    "],"
    "\"removed\": [],"
    "\"msg\": \"\","
    "\"timestamp\": \"$timestamp\","
    "\"time_str\": \"\""
"}"
)

nats request observations.down.tapir-pop "${json_msg[*]}"
