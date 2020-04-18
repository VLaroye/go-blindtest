package main

import (
	"math/rand"
	"sort"
	"time"

	"github.com/satori/go.uuid"
)

type SocketIOConnectedEvent struct {
	Game Game `json:"game_status"`
	Player Player `json:"player"`
}

type SocketIOSongStartedEvent struct {
	SongPreviewURI string `json:"preview_uri"`
}

type SocketIOArtistGuessedEvent struct {
	ArtistName string `json:"artist_name"`
	ArtistPictureURI string `json:"artist_picture_uri"`
}

type SocketIOSongGuessedEvent struct {
	SongTitle string `json:"song_title"`
}

type SocketIOUpdateEvent struct {
	Game Game `json:"game"`
}

type SocketIOResponseEvent struct {
	Song Song `json:"song"`
}

type Playlist struct {
	Songs []Song `json:"data"`
	Length int `json:"total"`
	// TODO: Think about other struct, 'next' is not meaningful here
	Next string `json:"next"`
}

func (p *Playlist) getRandomSong() Song {
	// Generate non-negative random number, between 0 and playlist's length + 1
	rand.Seed(time.Now().UnixNano())
	idx := rand.Intn(p.Length - 1)

	// Use random generated index to access a song in playlist
	return p.Songs[idx]
}

type Song struct {
	Id int `json:"id"`
	Preview string `json:"preview"`
	Artist Artist `json:"artist"`
	Title string `json:"title_short"`
}

type Artist struct {
	Name string `json:"name"`
	Picture string `json:"picture"`
}

type Game struct {
	Players []*Player
	CurrentRound Round
	SongsPlayed []Song
}

func newGame(players []*Player) Game {
	return Game{ Players: players, CurrentRound: Round{}}
}

func (g *Game) restart() {
	g.CurrentRound = Round{}
	g.SongsPlayed = make([]Song, 0)
	for _, v := range g.Players {
		v.resetScore()
	}
}

func (g *Game) join(player *Player) {
	g.Players = append(g.Players, player)
}

func (g *Game) leave(player *Player) {
	tempPlayers := make([]*Player, 0)

	for _, v := range g.Players {
		if v.ID != player.ID {
			tempPlayers = append(tempPlayers, v)
		}
	}

	g.Players = tempPlayers
}

func (g *Game) getLeaderBoard() *[]*Player {
	leaderBoard := make([]*Player, len(g.Players))
	copy(leaderBoard, g.Players)

	sort.SliceStable(leaderBoard, func(i, j int) bool {
		return leaderBoard[i].Score < leaderBoard[j].Score
	})

	return &leaderBoard
}

func (g *Game) getPlayerByID(id string) *Player {
	var foundPlayer Player
	for _, player := range g.Players {
		if player.ID.String() == id {
			foundPlayer = *player
		}
	}
	return &foundPlayer
}

func (g *Game) addSongToHistory(song *Song) {
	g.SongsPlayed = append(g.SongsPlayed, *song)
}

type Round struct {
	Nb int
	Song Song
	TimeLeft int
}

func (r *Round) countdown() {
	for range time.Tick(1 * time.Second) {
		r.TimeLeft--

		if r.TimeLeft == 0 {
			break
		}
	}
}

type Player struct {
	ID uuid.UUID `json:"id"`
	Name string `json:"name"`
	Score int `json:"score"`
}

func newPlayer(name string) *Player {
	return &Player{
		ID: uuid.Must(uuid.NewV4(), nil),
		Name: name,
		Score: 0,
	}
}

func (p *Player) resetScore() {
	p.Score = 0
}

func (p *Player) increaseScore(points int) {
	p.Score += points
}

