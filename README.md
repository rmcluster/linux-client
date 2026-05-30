# Linux Client

This is a simple client that announces itself to the server and runs an RPC server.

## Usage

You will need to have llama.cpp's rpc-server executable installed.

The following starts the Linux node software and connects it to the backend running on `127.0.0.1:4917`.
To connect to a different backend instance (ex. running on another computer), specify a different ip/port.

```bash
go run . -tracker 127.0.0.1:4917
```

## Docker

`compose.yml` provides an easy way to set up a single linux-client node on the device.
Note that if you are running the backend on the same device, you will need to use the `host.docker.internal` gateway.
This is configured by default for the compose.