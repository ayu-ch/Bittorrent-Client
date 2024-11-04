package torrent

import (
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/ayu-ch/bittorrent-client/pkg/bencode"
)

type Torrent struct {
	InfoHash [20]byte
	Info     Info
	Announce string
}

type Info struct {
	Name        string
	PieceLength int
	Pieces      [][20]byte
	Length      int
	Files       []File
}

type File struct {
	Length int
	Path   []string
}

// NewTorrent initializes a Torrent object from a .torrent file.
func NewTorrent(filename string) (*Torrent, error) {
	fileData, err := os.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to read torrent file: %w", err)
	}
	return NewTorrentFromBencode(fileData)
}

// NewTorrentFromBencode initializes a Torrent object from bencoded data.
func NewTorrentFromBencode(bencoded []byte) (*Torrent, error) {
	unmarshalledData, err := bencode.Unmarshal(bencoded)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal bencoded data: %w", err)
	}

	t := &Torrent{}
	for key, value := range unmarshalledData.(map[string]any) {
		switch key {
		case "info":
			t.Info = newInfo(value.(map[string]any))
		case "announce":
			t.Announce = value.(string)
		}
	}

	if err := t.updateInfoHash(); err != nil {
		return nil, fmt.Errorf("failed to update info hash: %w", err)
	}

	return t, nil
}

// newInfo constructs an Info object from bencoded data.
func newInfo(m map[string]any) Info {
	info := Info{}
	for key, value := range m {
		switch key {
		case "name":
			info.Name = value.(string)
		case "piece length":
			info.PieceLength = value.(int)
		case "pieces":
			piecesStr := value.(string)
			info.Pieces = make([][20]byte, len(piecesStr)/20)
			for i := 0; i < len(piecesStr); i += 20 {
				copy(info.Pieces[i/20][:], piecesStr[i:i+20])
			}
		case "length":
			info.Length = value.(int)
		case "files":
			for _, file := range value.([]any) {
				info.Files = append(info.Files, newFile(file.(map[string]any)))
			}
		}
	}
	return info
}

// newFile constructs a File object from bencoded data.
func newFile(m map[string]any) File {
	f := File{}
	for key, value := range m {
		switch key {
		case "length":
			f.Length = value.(int)
		case "path":
			for _, path := range value.([]any) {
				f.Path = append(f.Path, path.(string))
			}
		}
	}
	return f
}

// updateInfoHash calculates the SHA1 hash of the info dictionary.
func (t *Torrent) updateInfoHash() error {
	info := marshallableInfo(t.Info)
	infoBencoded, err := bencode.Marshal(info)
	if err != nil {
		return fmt.Errorf("failed to marshal info for hash: %w", err)
	}

	t.InfoHash = sha1.Sum(infoBencoded)
	return nil
}

// marshallableInfo prepares the Info object for bencoding.
func marshallableInfo(info Info) map[string]any {
	m := map[string]any{
		"name":         info.Name,
		"piece length": info.PieceLength,
		"pieces":       []byte{},
	}

	for _, piece := range info.Pieces {
		m["pieces"] = append(m["pieces"].([]byte), piece[:]...)
	}

	if len(info.Files) > 0 {
		m["files"] = []any{}
	} else {
		m["length"] = info.Length
	}

	for _, file := range info.Files {
		m["files"] = append(m["files"].([]any), marshallableFile(file))
	}
	return m
}

// marshallableFile prepares the File object for bencoding.
func marshallableFile(f File) map[string]any {
	m := map[string]any{
		"length": f.Length,
		"path":   []any{},
	}
	for _, path := range f.Path {
		m["path"] = append(m["path"].([]any), path)
	}
	return m
}

// buildTrackerURL constructs the tracker announce URL.
func (t *Torrent) buildTrackerURL(peerID [20]byte, port uint16) (string, error) {
	base, err := url.Parse(t.Announce)
	if err != nil {
		return "", fmt.Errorf("failed to parse announce URL: %w", err)
	}

	params := url.Values{
		"info_hash":  {string(t.InfoHash[:])},
		"peer_id":    {string(peerID[:])},
		"port":       {strconv.Itoa(int(port))},
		"uploaded":   {"0"},
		"downloaded": {"0"},
		"compact":    {"1"},
		"left":       {strconv.Itoa(t.Info.Length)}, // Total length of the file
	}

	base.RawQuery = params.Encode()
	return base.String(), nil
}

// AnnounceToTracker sends a GET request to the tracker to announce the peer.
func (t *Torrent) AnnounceToTracker(peerID [20]byte, port uint16) error {
	trackerURL, err := t.buildTrackerURL(peerID, port)
	if err != nil {
		return fmt.Errorf("failed to build tracker URL: %w", err)
	}

	resp, err := http.Get(trackerURL)
	if err != nil {
		return fmt.Errorf("failed to announce to tracker: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tracker returned non-200 status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read tracker response: %w", err)
	}

	return t.parseTrackerResponse(body)
}

// parseTrackerResponse parses the bencoded response from the tracker.
func (t *Torrent) parseTrackerResponse(data []byte) error {
	response, err := bencode.Unmarshal(data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal tracker response: %w", err)
	}

	trackerData := response.(map[string]any)

	// Print the entire response for debugging
	fmt.Printf("Raw tracker response: %+v\n", trackerData)

	// Extract interval
	if interval, ok := trackerData["interval"].(int); ok {
		fmt.Printf("Tracker interval: %d seconds\n", interval)
	} else {
		return fmt.Errorf("invalid or missing interval in tracker response")
	}

	// Extract peers
	if peersData, ok := trackerData["peers"]; ok {
		switch peers := peersData.(type) {
		case string:
			t.parsePeers(peers)
		default:
			return fmt.Errorf("invalid peers data type")
		}
	} else {
		return fmt.Errorf("missing peers in tracker response")
	}

	return nil
}

// parsePeers extracts IP addresses and ports from the binary blob of peers.
func (t *Torrent) parsePeers(peers string) {
	numPeers := len(peers) / 6 // Each peer is 6 bytes
	for i := 0; i < numPeers; i++ {
		peer := peers[i*6 : (i+1)*6]
		ip := fmt.Sprintf("%d.%d.%d.%d", peer[0], peer[1], peer[2], peer[3])
		port := (uint16(peer[4]) << 8) | uint16(peer[5])
		fmt.Printf("Peer: %s:%d\n", ip, port)
	}
}
