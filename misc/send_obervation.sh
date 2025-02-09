#!/bin/bash

nats request observations.down.tapir-pop "$(cat sample_observation.json)"
