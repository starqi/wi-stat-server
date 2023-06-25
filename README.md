Simple stat server (persisted data) and session/chat server (ephemeral data) for unnamed game project.

    TODO

        - Gin timeout
            - Test by turning off aggregator
        - Chat message size cap
            - Kick, and enforce same limit in UI
        - Rename chat to sessions
        - Different game servers + chat design
        - Unique key for key col
        - Redo proper full endpoint api_test, now using AES
        ? Set for deletion IDs
        - TODOs
        - Make Dockerfiles cache if no lib changes...
        - gorilla/websocket is unmaintained
        - Try gofmt
