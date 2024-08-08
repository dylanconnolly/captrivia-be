## Design Decisions

I took the approach of an event-driven design pattern which is commonly used for multiplayer games or messaging systems. The packages are broken into server (for handling http/websockets), redis (can be replaced with any datastore), and captrivia (business logic).

### Server
__Client:__ Represents an individual player or user, which connects to the system through a WebSocket connection. Each client is associated with a Hub (or GameHub), which manages its lifecycle and communication.


__Hub:__ Acts as a central communication router that registers clients and handles the dispatch of messages or events to connected clients. It manages the overall connection pool and ensures that messages reach their intended recipients. The Hub broadcasts global events such as player connect/disconnect as well as routing GameEvent changes to players that are not currently in a game such as a new game being created, the state of a game changing, or the number of players in a game changing.


__GameHub:__ A specialized hub focused on managing the state and interactions within a specific game. It handles game-specific logic such as player registration, broadcasting game events, processing commands, and managing game state transitions. It is designed to operate concurrently, handling multiple client connections and game events asynchronously.

This made the most sense in my head in terms of defining separation of concerns. Clients are registered to a Hub which manages all active clients. When a new game is created, the Hub initializes a new GameHub and transfers the creating client to that GameHub. Until the player leaves that game, the GameHub will be responsible for managing that client.

### Repository
I defined a GameService interface which allows any datastore to be dropped in as a replacement.

__Redis__ I used Redis as a very simple key-value store to store game state. Chose Redis mainly because I figured at scale there would be many reads/writes and Redis is great at handling those with low latency as well as its ability to handle a high volume of operations/sec. I had originally planned to used Redis' Pub/Sub as a means to broadcast events from the Hub or GameHub but didn't get around to implementing it. I also originally thought I'd want a way to recover from when the server crashed or a client disconnected in the middle of a game but obviously haven't implemented that as its a bit beyond the scope of the time I had.

### Business Logic
The captrivia package mainly houses business logic, such as the Game and Questions. I didn't get around to it but had I implemented player stat tracking, it too would have lived here.

## Benefits

This design approach allows for scalable game management, where each game instance can operate independently, making it easier to scale horizontally. From a development perspective, this approach enabled me to leverage goroutines and channels which made handling multiple clients and game events easier to manage. The modularity allows future changes to be easier to implement in my eyes.

