#!/bin/bash

# TODO check if bucket exists first

nats kv add observations

nats kv put observations tapircore.events.observations.com.example.suspicious $1
