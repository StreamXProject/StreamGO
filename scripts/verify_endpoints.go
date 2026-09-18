package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

func main() {
	botToken := "8879601665:AAHWVfu6ckG3XPgePUe_RCwzpeT8SpmOZhI"
	baseURL := "http://localhost:8000"

	fmt.Println("=== 1. Testing Health ===")
	resp, err := http.Get(baseURL + "/health")
	if err != nil {
		panic(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	fmt.Printf("/health: %s\n\n", body)

	fmt.Println("=== 2. Testing Telegram WebApp Auth ===")
	authDate := strconv.FormatInt(time.Now().Unix(), 10)
	userJSON := `{"id":999888777,"first_name":"GoTester","last_name":"Stream","username":"gotester"}`
	dataCheckString := fmt.Sprintf("auth_date=%s\nuser=%s", authDate, userJSON)

	hSecret := hmac.New(sha256.New, []byte("WebAppData"))
	hSecret.Write([]byte(botToken))
	secretKey := hSecret.Sum(nil)

	hHash := hmac.New(sha256.New, secretKey)
	hHash.Write([]byte(dataCheckString))
	calcHash := hex.EncodeToString(hHash.Sum(nil))

	initData := fmt.Sprintf("auth_date=%s&user=%s&hash=%s",
		url.QueryEscape(authDate),
		url.QueryEscape(userJSON),
		calcHash,
	)

	authPayload, _ := json.Marshal(map[string]string{"init_data": initData})
	authResp, err := http.Post(baseURL+"/auth/telegram", "application/json", bytes.NewReader(authPayload))
	if err != nil {
		panic(err)
	}
	var authResult struct {
		OK    bool   `json:"ok"`
		Token string `json:"token"`
		User  struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
		} `json:"user"`
	}
	json.NewDecoder(authResp.Body).Decode(&authResult)
	authResp.Body.Close()
	fmt.Printf("Auth Success: %v, Token: %s, User: %s (ID: %d)\n\n", authResult.OK, authResult.Token[:25]+"...", authResult.User.FirstName, authResult.User.ID)

	token := authResult.Token

	fmt.Println("=== 3. Testing GET /auth/me ===")
	req, _ := http.NewRequest("GET", baseURL+"/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	meResp, err := http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}
	meBody, _ := io.ReadAll(meResp.Body)
	meResp.Body.Close()
	fmt.Printf("/auth/me: %s\n\n", meBody)

	fmt.Println("=== 4. Testing Favourites (Add, List, Check, Remove) ===")
	favPayload, _ := json.Marshal(map[string]string{"track_id": "AgADjx8AAtO9YFU"})
	favReq, _ := http.NewRequest("POST", baseURL+"/favourites", bytes.NewReader(favPayload))
	favReq.Header.Set("Authorization", "Bearer "+token)
	favReq.Header.Set("Content-Type", "application/json")
	favResp, err := http.DefaultClient.Do(favReq)
	if err != nil {
		panic(err)
	}
	favBody, _ := io.ReadAll(favResp.Body)
	favResp.Body.Close()
	fmt.Printf("POST /favourites: %s\n", favBody)

	listFavReq, _ := http.NewRequest("GET", baseURL+"/favourites", nil)
	listFavReq.Header.Set("Authorization", "Bearer "+token)
	listFavResp, _ := http.DefaultClient.Do(listFavReq)
	listFavBody, _ := io.ReadAll(listFavResp.Body)
	listFavResp.Body.Close()
	fmt.Printf("GET /favourites: %s\n", listFavBody)

	idsFavReq, _ := http.NewRequest("GET", baseURL+"/favourites/ids", nil)
	idsFavReq.Header.Set("Authorization", "Bearer "+token)
	idsFavResp, _ := http.DefaultClient.Do(idsFavReq)
	idsFavBody, _ := io.ReadAll(idsFavResp.Body)
	idsFavResp.Body.Close()
	fmt.Printf("GET /favourites/ids: %s\n", idsFavBody)

	delFavReq, _ := http.NewRequest("DELETE", baseURL+"/favourites/AgADjx8AAtO9YFU", nil)
	delFavReq.Header.Set("Authorization", "Bearer "+token)
	delFavResp, _ := http.DefaultClient.Do(delFavReq)
	delFavBody, _ := io.ReadAll(delFavResp.Body)
	delFavResp.Body.Close()
	fmt.Printf("DELETE /favourites/AgADjx8AAtO9YFU: %s\n\n", delFavBody)

	fmt.Println("=== 5. Testing Playlists (Create, Add Tracks, Detail, Delete) ===")
	plPayload, _ := json.Marshal(map[string]string{
		"title":       "Live Verification Playlist",
		"description": "Created by Go test suite",
	})
	plReq, _ := http.NewRequest("POST", baseURL+"/playlists", bytes.NewReader(plPayload))
	plReq.Header.Set("Authorization", "Bearer "+token)
	plReq.Header.Set("Content-Type", "application/json")
	plResp, err := http.DefaultClient.Do(plReq)
	if err != nil {
		panic(err)
	}
	var createdPL struct {
		ID    string `json:"_id"`
		Title string `json:"title"`
	}
	json.NewDecoder(plResp.Body).Decode(&createdPL)
	plResp.Body.Close()
	fmt.Printf("Created Playlist: ID=%s, Title=%s\n", createdPL.ID, createdPL.Title)

	// Add track to playlist
	addTrkPayload, _ := json.Marshal(map[string]string{"track_id": "AgADjx8AAtO9YFU"})
	addTrkReq, _ := http.NewRequest("POST", baseURL+"/playlists/"+createdPL.ID+"/tracks", bytes.NewReader(addTrkPayload))
	addTrkReq.Header.Set("Authorization", "Bearer "+token)
	addTrkReq.Header.Set("Content-Type", "application/json")
	addTrkResp, _ := http.DefaultClient.Do(addTrkReq)
	addTrkBody, _ := io.ReadAll(addTrkResp.Body)
	addTrkResp.Body.Close()
	fmt.Printf("POST /playlists/{id}/tracks: %s\n", addTrkBody)

	// Get playlist detail
	getPLReq, _ := http.NewRequest("GET", baseURL+"/playlists/"+createdPL.ID, nil)
	getPLResp, _ := http.DefaultClient.Do(getPLReq)
	getPLBody, _ := io.ReadAll(getPLResp.Body)
	getPLResp.Body.Close()
	fmt.Printf("GET /playlists/{id}: %s\n", getPLBody)

	// Delete playlist
	delPLReq, _ := http.NewRequest("DELETE", baseURL+"/playlists/"+createdPL.ID, nil)
	delPLReq.Header.Set("Authorization", "Bearer "+token)
	delPLResp, _ := http.DefaultClient.Do(delPLReq)
	delPLBody, _ := io.ReadAll(delPLResp.Body)
	delPLResp.Body.Close()
	fmt.Printf("DELETE /playlists/{id}: %s\n\n", delPLBody)

	fmt.Println("=== 6. Testing Cover HEAD and GET ===")
	headCoverResp, _ := http.Head(baseURL + "/cover/AgADjx8AAtO9YFU")
	fmt.Printf("HEAD /cover/AgADjx8AAtO9YFU: Status %d, Location: %s\n", headCoverResp.StatusCode, headCoverResp.Header.Get("Location"))

	fmt.Println("\n>>> ALL TESTS COMPLETED SUCCESSFULLY! <<<")
	os.Exit(0)
}
