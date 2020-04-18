package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	socketio "github.com/mlsquires/socketio"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

const (
	playlistURI = "https://api.deezer.com/playlist/3775256682/tracks"
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
		var player *Player
		log.Printf("Socket %v connected", so.Id())

		so.Join("gameRoom")

		so.On("connectdd", func(params map[string]string) {
			playerName := params["player_name"]
			player = newPlayer(playerName)
			game.join(player)

			so.Emit("connectcc", SocketIOConnectedEvent{Game: game, Player: *player})

			log.Printf("%v joined the game", player.Name)
		})

		so.On("disconnect", func() {
			if player != nil {
				game.leave(player)
				log.Printf("%v (socket id: %v) left the game", player.Name, so.Id())
			}
		})

		so.On("guess", func(params map[string]string) {
			player.increaseScore(10)
			socketIOServer.BroadcastTo(
				"gameRoom",
				"update",
				SocketIOUpdateEvent{Game: game},
			)
			log.Printf("Guess received from: %v (socket id: %v). Guess: %v\n", player.ID, so.Id(), params["guess"])

			songGuessed, artistGuessed := handleGuess(game.CurrentRound.Song, params["guess"])

			if artistGuessed {
				player.increaseScore(10)
				so.Emit("artistGuessed", SocketIOArtistGuessedEvent{ArtistName: game.CurrentRound.Song.Artist.Name, ArtistPictureURI: game.CurrentRound.Song.Artist.Picture})
				socketIOServer.BroadcastTo(
					"gameRoom",
					"update",
					SocketIOUpdateEvent{Game: game},
				)
			}

			if songGuessed {
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

func startGame() {
	for {
		roundNb := game.CurrentRound.Nb + 1
		round := Round{
			Nb:       roundNb,
			Song:     playlist.getRandomSong(),
			TimeLeft: 30,
		}
		game.CurrentRound = round

		// Send 'song' message with song details
		log.Println(game.CurrentRound.Song.Preview, game.CurrentRound.Song.Title, game.CurrentRound.Song.Artist)
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
	socketIOServer.BroadcastTo("gameRoom","gameFinished", game.getLeaderBoard())
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

func handleGuessEvent(so socketio.Socket, playerId, guess string) {
	player := game.getPlayerByID(playerId)

	songGuessed, artistGuessed := handleGuess(game.CurrentRound.Song, guess)

	if artistGuessed {
		player.increaseScore(10)
		so.Emit("artistGuessed", SocketIOArtistGuessedEvent{ArtistName: game.CurrentRound.Song.Artist.Name, ArtistPictureURI: game.CurrentRound.Song.Artist.Picture})
		socketIOServer.BroadcastTo(
			"gameRoom",
			"update",
			SocketIOUpdateEvent{Game: game},
		)
	}

	if songGuessed {
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

func handleGuess(song Song, guess string) (bool, bool){
	sanitizedGuess := sanitizeString(guess)
	sanitizedSongTitle := sanitizeString(song.Title)
	sanitizedSongArtist := sanitizeString(song.Artist.Name)

	return sanitizedGuess == sanitizedSongTitle, sanitizedGuess == sanitizedSongArtist
}

func sanitizeString(s string) string {
	sanitizedString := s
	// Lowercase
	sanitizedString = strings.ToLower(sanitizedString)
	// Remove spaces
	sanitizedString = strings.ReplaceAll(sanitizedString, " ", "")
	// Remove accents
	sanitizedString = removeAccents(sanitizedString)
	// Remove special chars
	sanitizedString = removeNonAlphanumericChars(sanitizedString)

	return sanitizedString
}


func removeAccents(s string) string  {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	output, _, e := transform.String(t, s)
	if e != nil {
		panic(e)
	}
	return output
}

func removeNonAlphanumericChars(s string) string {
	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9]+")
	if err != nil {
		log.Fatal(err)
	}
	return reg.ReplaceAllString(s, "")
}