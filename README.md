# Example usage
Coming soon...

# Sample config
```yaml
# Enable debug output
Debug:

# Url of the MQTT broker
MqttUrl:

# Root certificate for the MQTT mTLS PKI
MqttCaCert:

# Client certificate for authentication when connecting to the MQTT broker
MqttClientCert:

# Private key for authentication when connecting to the MQTT broker
MqttClientKey:

# Enable/disable TLS keylogfile (also need to set path in env variable)
MqttEnableTlsKlf:

# URL of the NATS server
NatsUrl:

# URL of the Nodeman API (only used by upbound bridges)
NodemanApiUrl:

# List of bridges
Bridges:
      # Direction to bridge in, MQTT->NATS (up) or NATS->MQTT (down)
    - Direction:

      # MQTT topic used by bridge (wildcards possible for "up" bridges)
	  MqttTopic:

      # NATS subject used by bridge (wildcards possible for "down" bridges)
	  NatsSubject:

      # NATS queue group for load balancing (only used for "down" bridges)
	  NatsQueue:

      # Key to sign (downbound bridges) or validate (upbound bridges) data
      # Upbound bridges can also use the Nodeman API to fetch validation keys
	  Key:

      # Schema to validate data against
	  Schema:

    - Direction:
	  MqttTopic:
	  NatsSubject:
	  NatsQueue:
	  Key:
	  Schema:
```
