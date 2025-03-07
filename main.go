package main

import (
	"bytes"
	"context"
	"sync"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	API_URL = "https://randomuser.me/api/"
	TOTAL_RECORDS = 15000
	PER_PAGE = 3750
	CONCURRENCY = 4
	REDIS_USERS_KEY = "users"
)

type UsersResults struct {
	Results []struct {
		Gender string `json:"gender"`
		Name struct {
			First string `json:"first"`
			Last string `json:"last"`
		} `json:"name"`
		Email string `json:"email"`
		Location struct {
			City string `json:"city"`
			Country string `json:"country"`
		} `json:"location"`
		Login struct {
			Uuid string `json:"uuid"`
		} `json:"login"`
	}
}

type User struct {
	Gender string `json:"gender"`
	FirstName string `json:"first_name"`
	LastName string `json:"last_name"`
	Email string `json:"email"`
	City string `json:"city"`
	Country string `json:"country"`
	Uuid string `json:"uuid"`
}

func fetchUser(count int, page int) (UsersResults, error) {
	urlToFetch := fmt.Sprintf("%s?results=%d&page=%d&inc=gender,name,location,login&noinfo", API_URL, count, page)

	resp, err := http.Get(urlToFetch)

	if err != nil {
		fmt.Printf("Error fetching URL: %v\n", err)

		return UsersResults{}, err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Unexpected status code: %d\n", resp.StatusCode)

		return UsersResults{}, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)

	if err != nil {
		fmt.Printf("Error fetching URL: %v\n", err)

		return UsersResults{}, err
	}

	var users UsersResults

	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&users); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)

		return UsersResults{}, err
	}

	return users, nil
}

func fetchUsersAsync(wg *sync.WaitGroup, ch chan<- UsersResults, count int, page int) {
	defer wg.Done()

	users, err := fetchUser(count, page)

	if err != nil {
		fmt.Printf("Error fetching the user: %v\n", err)

		return
	}

	ch <- users
}

func fetchAllUsers() []User {
	wg := sync.WaitGroup{}
	ch := make(chan UsersResults, CONCURRENCY)
	pages := TOTAL_RECORDS / PER_PAGE

	for i := 0; i < pages; i++ {
		wg.Add(1)

		go fetchUsersAsync(&wg, ch, PER_PAGE, i + 1)
	}

	go func () {
		wg.Wait()
		close(ch)
	}()

	var results []User

	for usersResults := range ch {
		for _, user := range usersResults.Results {
			results = append(
				results,
				User{
					Gender: user.Gender,
					FirstName: user.Name.First,
					LastName: user.Name.Last,
					Email: user.Email,
					City: user.Location.City,
					Country: user.Location.Country,
					Uuid: user.Login.Uuid,
				},
			)
		}
	}

	return results
}

func encodeUsers(users []User) string {
	encodedUsers, err := json.Marshal(users)

	if err != nil {
		fmt.Printf("Error encoding users: %v\n", err)

		return ""
	}

	return string(encodedUsers)
}

// Redis
var ctx = context.Background()

func getRedisConnection() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		Password: "",
		DB: 0,
	})
}

func fetchFromRedis(rdb *redis.Client) (string, error) {
	val, err := rdb.Get(ctx, REDIS_USERS_KEY).Result()

	if err != nil {
		return "", err
	}

	return val, nil
}

func storeInRedis(rdb *redis.Client, val string) error {
	return rdb.Set(ctx, REDIS_USERS_KEY, val, 0).Err()
}

func getUsers(c *gin.Context) {
	rdb := getRedisConnection()

	var output string
	var err error

	output, err = fetchFromRedis(rdb)

	if err == redis.Nil {
		results := fetchAllUsers()
		output = encodeUsers(results)

		err2 := storeInRedis(rdb, output)

		if err2 != nil {
			c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Error storing users in redis"})

			return
		}
	} else if err != nil {
		fmt.Printf("Error fetching users from redis: %v\n", err)

		c.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "Error fetching users from redis"})

		return
	}

	c.String(http.StatusOK, output)
}

func main() {
	router := gin.Default()

	router.GET("/users", getUsers)

	router.Run("localhost:8080")
}
