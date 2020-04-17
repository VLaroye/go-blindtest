package main

import (
	"encoding/json"
	"github.com/gin-gonic/gin"
	socketio "github.com/googollee/go-socket.io"
	"log"
	"net/http"
	"reflect"
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
var currentSong Song

func main() {
	var err error
	router := initRouter()

	socketIOServer, err = socketio.NewServer(nil)
	if err != nil {
		log.Fatal(err)
	}

	socketIOServer.OnConnect("/", func(s socketio.Conn) error {
		newPlayer := newPlayer("Agathe")
		game.join(newPlayer)

		s.SetContext(newPlayer.ID.String())

		s.Join("gameRoom")
		s.Join("player_" + newPlayer.ID.String())

		s.Emit("connected", newPlayer.ID.String(), game, currentSong)

		return nil
	})

	socketIOServer.OnDisconnect("/", func(s socketio.Conn, reason string) {
		playerId := reflect.ValueOf(s.Context()).String()
		player := game.getPlayerByID(playerId)
		game.leave(player)
	})

	playlist = *getPlaylist(playlistURI)
	game = newGame(make([]Player, 0))

	go startGame()

	go socketIOServer.Serve()
	defer socketIOServer.Close()

	router.GET("game/*any", gin.WrapH(socketIOServer))
	router.POST("game/*any", gin.WrapH(socketIOServer))


	err = router.Run()

	if err != nil {
		log.Fatalf("Error running app. Err: %v", err)
	}
}

func startGame() {
	for {
		game.CurrentRound++
		// Send 'song' message with song details
		currentSong = playlist.getRandomSong()
		log.Println(currentSong.Preview, currentSong.Title, currentSong.Artist)
		socketIOServer.BroadcastToRoom("", "gameRoom", "song", currentSong.Preview)

		// While waiting, handle 'guess' events
		socketIOServer.OnEvent("", "guess", func(s socketio.Conn, playerId string, guess string) {
			player := game.getPlayerByID(playerId)

			songGuessed, artistGuessed := handleGuess(currentSong, guess)

			if artistGuessed {
				player.increaseScore(10)
				socketIOServer.BroadcastToRoom("", "player_" + playerId , "artistGuessed", currentSong.Artist.Name, currentSong.Artist.Picture)
			}
			if songGuessed {
				player.increaseScore(10)
				socketIOServer.BroadcastToRoom("", "player_" + playerId , "songGuessed", currentSong.Title)
			}
		})

		// Wait 30 sec (song preview duration)
		time.Sleep(30 * time.Second)

		// TODO: Workaround, maybe there is a better to stop listening to "guess" events
		socketIOServer.OnEvent("", "guess", func(s socketio.Conn, guess string) {
			return
		})

		// After 30sec, send artist + title
		socketIOServer.BroadcastToRoom("", "gameRoom", "response", currentSong.Artist, currentSong.Title)

		// Wait 10 sec before running new round
		time.Sleep(10 * time.Second)

		// TODO: Put nb of round into constant
		if game.CurrentRound == 10 {
			endGame()
		}
	}
}

func endGame() {
	socketIOServer.BroadcastToRoom("", "gameRoom", "gameFinished", game.getLeaderBoard())
	game.restart()
}

func getPlaylist(URI string) *Playlist {
	var playlist Playlist

	response, err := http.Get(URI)
	if err != nil {
		log.Printf("Can't get song. Err: %v", err.Error())
		return nil
	}

	json.NewDecoder(response.Body).Decode(&playlist)

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