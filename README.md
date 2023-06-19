Simple stat server (persisted data) and session/chat server (ephemeral data) for unnamed game project.

    TODO

        / Chat -> chat, session server
        / Extract chat
        - Gin timeout
            - Test by turning off aggregator
        - Provide in game name and actually use it
        - Channel closes?
        - Rename chat to sessions
        - Expiry mechanism
            - Lock clean up if in game
            - Re-request if expired otherwise
        - Different game servers + chat design
        - CORS design chat
        - CORS design stats
        - Prod build Gin
        - Unique key for key col
        - Redo proper full endpoint api_test, now using AES
        ? Set for deletion IDs
        - TODOs
        - Make Dockerfiles cache if no lib changes...
        - Review chat code and add more proper design & security
        - gorilla/websocket is unmaintained
