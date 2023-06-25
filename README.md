Simple stat server (persisted data) and session/chat server (ephemeral data) for unnamed game project.

    TODO

        - Gin timeout
            - Test by turning off aggregator
        - Chat message size cap
        - Rename chat to sessions
        - Different game servers + chat design
        - Prod build Gin
        - Unique key for key col
        - Redo proper full endpoint api_test, now using AES
        - GUID collision just in case
        ? Set for deletion IDs
        - TODOs
        - Make Dockerfiles cache if no lib changes...
        - gorilla/websocket is unmaintained
        - Try gofmt
