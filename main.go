package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	"github.com/mlsquires/socketio"
	"log"
	"net/http"
	"time"
)

const (
	playlistURI = "https://api.deezer.com/playlist/7530596462/tracks"
)

var socketIOServer *socketio.Server
var game Game
var playlist Playlist

func main() {
	var err error
	router := initRouter()

	socketIOServer, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	socketIOServer.On("connection", func(so socketio.Socket) {
		log.Printf("Socket %v connected", so.Id())

		so.Join("gameRoom")

		so.On("join", func(params map[string]string) {
			playerName, ok := params["player_name"]

			if !ok {
				so.Emit("error", "Field 'player_name' required")
				return
			}

			handleJoinEvent(so, playerName)
		})

		so.On("playerReconnect", func(params map[string]string) {
			playerID, ok := params["player_id"]

			if !ok {
				so.Emit("error", "Field 'player_id' required")
				return
			}

			handleReconnectEvent(so, playerID)
		})

		so.On("disconnect", func() {
			log.Printf("Socket id: %v) left the game", so.Id())
		})

		so.On("guess", func(params map[string]string) {
			playerID, ok := params["player_id"]
			playerGuess, ok := params["guess"]

			if !ok {
				so.Emit("error", "Field 'player_id' and 'guess' required")
				return
			}

			handleGuessEvent(so, playerID, playerGuess)
		})
	})

	socketIOServer.On("error", func(so socketio.Socket, err error) {
		log.Printf("Error: %v", err.Error())
	})

	playlist = *getPlaylist(playlistURI)
	game = newGame(make([]*Player, 0))

	go startGame()

	router.GET("game/*any", gin.WrapH(socketIOServer))
	router.POST("game/*any", gin.WrapH(socketIOServer))

	err = router.Run()

	if err != nil {
		log.Fatalf("Error running app. Err: %v", err)
	}
}

func handleJoinEvent(so socketio.Socket, playerName string) {
	player := newPlayer(playerName)
	game.join(player)

	so.Emit("joined", SocketIOConnectedEvent{Game: game, Player: *player})

	log.Printf("%v joined the game", player.Name)
}

func handleReconnectEvent(so socketio.Socket, playerID string) {
	player, err := game.getPlayerByID(playerID)
	log.Println(playerID)
	if err != nil {
		so.Emit("error", err)
	}
	so.Emit("joined", SocketIOConnectedEvent{Game: game, Player: *player})
}

func handleGuessEvent(so socketio.Socket, playerID, playerGuess string) {
	player, err := game.getPlayerByID(playerID)

	if err != nil {
		so.Emit("err", err)
	}

	guess := newGuess(playerGuess, game.CurrentRound.Song)

	log.Printf("Guess received from: %v. Guess: %v\n", player.ID, playerGuess)

	if guess.artistGuessed() {
		player.increaseScore(10)
		so.Emit("artistGuessed", SocketIOArtistGuessedEvent{ArtistName: game.CurrentRound.Song.Artist.Name})
		socketIOServer.BroadcastTo(
			"gameRoom",
			"update",
			SocketIOUpdateEvent{Game: game},
		)
	}

	if guess.songGuessed() {
		player.increaseScore(10)
		so.Emit(
			"songGuessed",
			SocketIOSongGuessedEvent{SongTitle: game.CurrentRound.Song.Title},
		)
		socketIOServer.BroadcastTo(
			"gameRoom",
			"update",
			SocketIOUpdateEvent{Game: game},
		)
	}
}

func startGame() {
	for {
		roundNb := game.CurrentRound.Nb + 1
		round := Round{
			Nb:       roundNb,
			Song:     playlist.getRandomSong(),
			TimeLeft: 30,
		}
		game.CurrentRound = round
		log.Printf("Round %v started. Song: %v - %v", game.CurrentRound.Nb, game.CurrentRound.Song.Title, game.CurrentRound.Song.Artist.Name)
		// Send 'song' message with song details
		socketIOServer.BroadcastTo(
			"gameRoom",
			"songStarted",
			SocketIOSongStartedEvent{SongPreviewURI: game.CurrentRound.Song.Preview},
		)

		// Wait 30 sec (song preview duration)
		game.CurrentRound.countdown()

		// After 30sec, send artist + title
		socketIOServer.BroadcastTo(
			"gameRoom",
			"response",
			SocketIOResponseEvent{Song: game.CurrentRound.Song},
		)
		game.addSongToHistory(&game.CurrentRound.Song)

		// Wait 10 sec before running new round
		time.Sleep(10 * time.Second)

		// TODO: Put nb of round into constant
		if game.CurrentRound.Nb == 10 {
			endGame()
		}
	}
}

func endGame() {
	socketIOServer.BroadcastTo("gameRoom", "gameFinished", game.getLeaderBoard())
	game.restart()
}

func getPlaylist(URI string) *Playlist {
	var playlist Playlist

	response, err := http.Get(URI)
	if err != nil {
		log.Printf("Can't get song. Err: %v", err.Error())
		return nil
	}

	err = json.NewDecoder(response.Body).Decode(&playlist)
	if err != nil {
		log.Fatalf("Couldn't decode playlsit JSON. Err: %v", err)
	}

	if playlist.Next != "" {
		tempPlaylist := getPlaylist(playlist.Next)

		playlist.Songs = append(playlist.Songs, tempPlaylist.Songs...)
	}

	playlist.Songs = *filterSongsWithoutPreview(&playlist.Songs)
	playlist.Length = len(playlist.Songs)

	return &playlist
}

func filterSongsWithoutPreview(songs *[]Song) *[]Song {
	tempSlice := make([]Song, 0)

	for _, v := range *songs {
		if v.Preview != "" {
			tempSlice = append(tempSlice, v)
		}
	}

	return &tempSlice
}
