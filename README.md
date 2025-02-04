# Start NATS server (requires docker)
```
./misc/start_server.sh
```

# Install the NATS CLI
```
go install github.com/nats-io/natscli/nats@latest
```

# Initialize a bucket and store some data (requires NATS CLI)
```
./misc/send_message.sh example.com 256
```

# Build the mqtt-sender
```
make build
```

# Start the mqtt-sender
```
./bin/mqtt-sender
```

# Update the data in the bucket and trigger a message
```
./misc/send_message.sh example.com 257
```

