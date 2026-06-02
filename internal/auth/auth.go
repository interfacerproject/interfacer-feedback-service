package auth

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

//go:embed zenflows-crypto/src/verify_graphql.zen
var VERIFY string

// zenroomMutex serializes access to Zenroom - the library is not thread-safe
var zenroomMutex sync.Mutex

// ZenResult mirrors the zenroom binding result
type ZenResult struct {
	Output string
	Logs   string
}

// Input and output of sign_graphql.zen
type ZenroomData struct {
	Gql            string `json:"gql"`
	EdDSASignature string `json:"eddsa_signature"`
	EdDSAPublicKey string `json:"eddsa_public_key"`
}

type ZenroomResult struct {
	Output []string `json:"output"`
}

// zencodeExec runs zencode-exec with proper pipe handling to avoid deadlock
// on large inputs (>4KB due to pipe buffer size)
func zencodeExec(script, conf, keys, data string) (ZenResult, bool) {
	cmd := exec.Command("zencode-exec")

	// Create pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return ZenResult{Logs: fmt.Sprintf("stdin pipe error: %v", err)}, false
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Start the command BEFORE writing to stdin to avoid deadlock
	if err := cmd.Start(); err != nil {
		return ZenResult{Logs: fmt.Sprintf("start error: %v", err)}, false
	}

	// Write input in a goroutine to avoid blocking
	go func() {
		defer stdin.Close()

		// Format: conf\nscript_b64\nkeys_b64\ndata_b64\nextra_b64\ncontext_b64\n
		io.WriteString(stdin, conf)
		io.WriteString(stdin, "\n")

		io.WriteString(stdin, base64.StdEncoding.EncodeToString([]byte(script)))
		io.WriteString(stdin, "\n")

		io.WriteString(stdin, base64.StdEncoding.EncodeToString([]byte(keys)))
		io.WriteString(stdin, "\n")

		io.WriteString(stdin, base64.StdEncoding.EncodeToString([]byte(data)))
		io.WriteString(stdin, "\n")

		io.WriteString(stdin, "") // extra
		io.WriteString(stdin, "\n")

		io.WriteString(stdin, "") // context
		io.WriteString(stdin, "\n")
	}()

	// Wait for command to complete
	err = cmd.Wait()

	return ZenResult{
		Output: stdout.String(),
		Logs:   stderr.String(),
	}, err == nil
}

func (data *ZenroomData) VerifyDid() error {
	baseUrl := os.Getenv("BASE_DID_URL")
	if baseUrl == "" {
		baseUrl = "https://explorer.did.dyne.org/details/"
	}
	context := os.Getenv("DID_CONTEXT_PATH")
	if context == "" {
		context = "did:dyne:ifacer"
	}
	url := fmt.Sprintf("%s%s:%s", baseUrl, context, data.EdDSAPublicKey)
	log.Printf("Fetching %s\n", url)
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("problem fetching DID, status: %d", resp.StatusCode)
	}
	return nil
}

func (data *ZenroomData) IsAuth() error {
	jsonData, _ := json.Marshal(data)

	log.Printf("Calling Zenroom with data size: %d bytes, gql size: %d bytes\n", len(jsonData), len(data.Gql))

	// Serialize Zenroom calls - the library is not thread-safe
	log.Println("Waiting for Zenroom mutex...")
	zenroomMutex.Lock()
	defer zenroomMutex.Unlock()
	log.Println("Acquired Zenroom mutex, executing...")

	log.Printf("VERIFY script length: %d bytes", len(VERIFY))
	log.Println("Calling zencodeExec now...")

	startTime := time.Now()
	result, success := zencodeExec(VERIFY, "", "", string(jsonData))
	log.Printf("Zenroom execution took %v", time.Since(startTime))

	if !success {
		log.Printf("Zenroom logs: %s", result.Logs)
		return errors.New(result.Logs)
	}

	var zenroomResult ZenroomResult
	err := json.Unmarshal([]byte(result.Output), &zenroomResult)
	if err != nil {
		log.Printf("Error unmarshaling zenroom output: %v", err)
		return err
	}
	if len(zenroomResult.Output) == 0 || zenroomResult.Output[0] != "1" {
		log.Printf("Signature not authentic, output: %v", zenroomResult.Output)
		return errors.New("signature is not authentic")
	}
	log.Println("Signature verified successfully")
	return nil
}
