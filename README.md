# Notification Service
Stream event, real-time for notification using SSE.
Using grpc protocol for push message to client.

# How to run

### Environment:
    Golang: [Download here!](https://golang.org/dl/)
    Port:
        1. gRPC: 55100
        2. SSE: 55102

### Setup:
    clone project
    $ cd notification-server
    $ make set-up

### Build:  
    $ make build

### Run:
    $ make run-local

# Example
### Client register
    # for now: token is clientID registered 
    $ curl http://localhost:55102?token=123

### Server push
    # run test push message to cliet
    $ go test ./test/ -count=1 -v -run TestPushMessage