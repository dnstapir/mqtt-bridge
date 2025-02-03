#!/bin/bash

# TODO check if bucket exists first

nats kv add observations

nats kv put observations tapir.core.events.observations.$1 $2
