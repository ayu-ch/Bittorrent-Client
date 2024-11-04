package main

import (
	"crypto/rand"
	"log"
	"os"

	// "github.com/ayu-ch/bittorrent-client/pkg/bencode"
	"github.com/ayu-ch/bittorrent-client/torrent"
)

func generatePeerID() ([20]byte, error) {
	var peerID [20]byte
	_, err := rand.Read(peerID[:])
	if err != nil {
		return [20]byte{}, err
	}
	return peerID, nil
}

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Torrent filename not provided as a command-line argument.")
		return
	}

	torrentFile := os.Args[1]

	// Initialize Torrent from the .torrent file
	torrentObj, err := torrent.NewTorrent(torrentFile)
	if err != nil {
		log.Fatalf("Failed to create Torrent object: %v", err)
		return
	}

	// fmt.Printf("The unmarshalled torrent file is: \n %+v \n", torrentObj)

	// Generate a random peer ID
	peerID, err := generatePeerID()
	if err != nil {
		log.Fatalf("Failed to generate peer ID: %v", err)
		return
	}

	// Example port
	port := uint16(6881)

	// Announce to the tracker
	if err := torrentObj.AnnounceToTracker(peerID, port); err != nil {
		log.Fatalf("Failed to announce to tracker: %v", err)
		return
	}
}
