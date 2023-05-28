package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	ice "github.com/pion/ice/v2"
)

type ICE struct {
	Username   string   `json:"username"`
	Password   string   `json:"password"`
	Candidates []string `json:"candidates"`
	mutex      sync.Mutex
}

func (i *ICE) Add(c ice.Candidate) {
	i.mutex.Lock()
	defer i.mutex.Unlock()

	i.Candidates = append(i.Candidates, c.Marshal())
}

func main() {
	if len(os.Args) != 2 {
		panic("ice send/receive")
	}

	var candidates *ICE = &ICE{}
	iceAgent, err := ice.NewAgent(&ice.AgentConfig{
		NetworkTypes: []ice.NetworkType{ice.NetworkTypeUDP4},
	})
	if err != nil {
		panic(err)
	}

	// When we have gathered a new ICE Candidate send it to the remote peer
	if err = iceAgent.OnCandidate(func(c ice.Candidate) {
		if c == nil {
			return
		}
		candidates.Add(c)
		fmt.Printf("candidate: (%s)\n", c.Marshal())
	}); err != nil {
		panic(err)
	}

	// When ICE Connection state has change print to stdout
	if err = iceAgent.OnConnectionStateChange(func(c ice.ConnectionState) {
		fmt.Printf("ICE Connection State has changed: %s\n", c.String())
	}); err != nil {
		panic(err)
	}

	// Get the local auth details and send to remote peer
	candidates.Username, candidates.Password, err = iceAgent.GetLocalUserCredentials()
	if err != nil {
		panic(err)
	}

	if err = iceAgent.GatherCandidates(); err != nil {
		panic(err)
	}

	time.Sleep(time.Millisecond * 100)
	remoteCandidates, err := encodeAndDecode(candidates)
	if err != nil {
		panic(err)
	}

	for _, c := range remoteCandidates.Candidates {
		rc, err := ice.UnmarshalCandidate(c)
		if err != nil {
			panic(err)
		}
		fmt.Println("Received request for: ")
		if err := iceAgent.AddRemoteCandidate(rc); err != nil {
			panic(err)
		}

	}

	var exec func(*ice.Agent, string, string) error

	if os.Args[1] == "send" {
		exec = Dial
	} else {
		exec = Accept
	}
	if err := exec(iceAgent, remoteCandidates.Username, remoteCandidates.Password); err != nil {
		panic(err)
	}
}

func Dial(iceAgent *ice.Agent, username, password string) error {
	conn, err := iceAgent.Dial(context.TODO(), username, password)
	if err != nil {
		return err
	}
	fmt.Println("Started communication with ", conn.RemoteAddr())
	go func() {
		for i := 0; i < 3; i++ {
			msg := fmt.Sprintf("hello %d", i)
			fmt.Fprint(conn, msg)
			fmt.Println("Sent: ", msg)
		}
	}()

	buf := make([]byte, 1500)
	for i := 0; i < 3; i++ {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}

		fmt.Printf("Received: '%s'\n", string(buf[:n]))
	}
	return nil
}

func Accept(iceAgent *ice.Agent, username, password string) error {
	conn, err := iceAgent.Accept(context.TODO(), username, password)
	if err != nil {
		return err
	}
	fmt.Println("Started communication with ", conn.RemoteAddr())
	go func() {
		for i := 0; i < 3; i++ {
			msg := fmt.Sprintf("world %d", i)
			fmt.Fprint(conn, msg)
			fmt.Println("Sent: ", msg)
		}
	}()

	buf := make([]byte, 1500)
	for i := 0; i < 3; i++ {
		n, err := conn.Read(buf)
		if err != nil {
			return err
		}

		fmt.Printf("Received: '%s'\n", string(buf[:n]))
	}
	return nil
}

func encodeAndDecode(candidates *ICE) (*ICE, error) {
	file, err := os.Create(os.Args[1] + ".json")
	if err != nil {
		return nil, err
	}
	if err := json.NewEncoder(file).Encode(&candidates); err != nil {
		return nil, err
	}
	var filename string
	if os.Args[1] == "send" {
		filename = "receive.json"
	} else {
		filename = "send.json"
	}
	file.Close()

	fmt.Printf("Waiting for file[%s]...", filename)

	for {
		file, err = os.Open(filename)
		if err == nil {
			fmt.Println("File detected")
			break
		}
		time.Sleep(time.Millisecond * 100)
	}

	remoteCandidates := &ICE{}
	if err := json.NewDecoder(file).Decode(remoteCandidates); err != nil {
		return nil, err
	}

	return remoteCandidates, nil
}
